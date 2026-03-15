<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:cat="http://www.w3.org/2012/10/xslt-test-catalog"
  xmlns:err="http://www.w3.org/2005/xqt-errors"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:rng="http://relaxng.org/ns/structure/1.0"
  exclude-result-prefixes="xs cat err rng">
  
  <xsl:output indent="yes" />
  
  <!-- Test that all non-error stylesheets are valid against the
       RelaxNG schema for stylesheets -->
  
  <xsl:template name="xsl:initial-template" match="/" use-when="function-available('rng:schema-report', 1)">
    <xsl:message>Performing RelaxNG Schema testing. RNG XPath extension Function is available.</xsl:message>
    <xsl:variable
      name="report-generator"
      select="rng:schema-report(resolve-uri('schema-for-xslt30.rnc', static-base-uri()))" />

    <out>
      
      <xsl:for-each
        select="/*/cat:test-set/document(@file)
        //cat:stylesheet[
          not(ancestor::cat:test-case//cat:error) and
          not(ancestor::cat:test-set/cat:dependencies/cat:feature/@value = 'XML_1.1')
        ]/@file">

        <xsl:message>RNG Validating <xsl:value-of select="."/></xsl:message>
        <xsl:variable name="doc" select="document(.)"/>

        <xsl:choose>
          <xsl:when
            test="not($doc/xsl:*) or 
                           ($doc//@version/number() > 3) or 
                           ($doc//@use-when) or ($doc//@xsl:use-when) or
                           ($doc//@Q{http://www.w3.org/2001/XMLSchema-instance}type) or
                           $doc//xs:*[not(ancestor::xsl:import-schema)]">
            <!-- don't attempt to validate documents rooted at an LRE, or those in forwards compatibility mode, 
                    or using use-when, or using xsi:type, or creating elements in the schema namespace -->
            <skip file="{.}"/>
          </xsl:when>
          <xsl:otherwise>

            <xsl:apply-templates
              select="$report-generator($doc)">
              <xsl:with-param name="uri" as="xs:string" select="."/>
              <xsl:with-param name="doc" as="document-node()" select="$doc" />
            </xsl:apply-templates>
            
          </xsl:otherwise>
        </xsl:choose>
      </xsl:for-each>
    </out>
  </xsl:template>
  
  <xsl:template name="xsl:initial-template" match="/" use-when="not(function-available('rng:schema-report', 1))">
    <xsl:message>Skipping RelaxNG Schema testing. RNG XPath extension Function is unavailable.</xsl:message>
  </xsl:template>
  
  <xsl:template match="report">
    <xsl:param name="uri" as="xs:string" />
    <xsl:param name="doc" as="document-node()" />
    
    <xsl:choose>
      <xsl:when test="count(fatal|error) eq 0">
        <ok file="{$uri}" size="{count($doc//*)}"/>
      </xsl:when>
      <xsl:otherwise>
        <xsl:message select="(fatal|error)[1]/message[1]"> ** Invalid ** </xsl:message>
        <invalid file="{$uri}" message="{(fatal|error)[1]/message[1]}" />
      </xsl:otherwise>
    </xsl:choose>
  </xsl:template>

</xsl:stylesheet>
