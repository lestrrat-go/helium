<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:cat="http://www.w3.org/2012/10/xslt-test-catalog"
  xmlns:err="http://www.w3.org/2005/xqt-errors"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:fn="http://www.w3.org/2005/xpath-functions"
  exclude-result-prefixes="#all">
  
  <xsl:param name="supports_xml1.1" static="true" as="xs:boolean" select="true()"/>
  
  <!-- Test that all non-error stylesheets are valid against the schema for stylesheets -->

  <xsl:import-schema namespace="http://www.w3.org/2012/10/xslt-test-catalog"/>

  <xsl:template match="/">
      <xsl:variable name="results" as="element()*">
        <xsl:for-each select="/*/cat:test-set/document(@file)//(cat:stylesheet|cat:package)[not(ancestor::cat:test-case//cat:error)]/@file">
          <!--<xsl:copy-of select="document(.)" validation="strict"/>-->
          <xsl:try>
            
            <xsl:variable name="doc" select="document(.)"/>
            <xsl:choose>
              <xsl:when test="not($doc/xsl:*) or 
                             ($doc//@version/number() > 3) or
                             ($doc//@xsl:version/number() > 3) or
                             ($doc//@use-when) or ($doc//@xsl:use-when) or
                             ($doc//@Q{http://www.w3.org/2001/XMLSchema-instance}type) or
                             $doc//xs:*[not(ancestor::xsl:import-schema)] or
                             $doc//fn:* or
                             exists(preceding::*/(cat:stylesheet|cat:package)/@file[. = current()])">
                 <!-- don't attempt to validate documents rooted at an LRE, or those in forwards compatibility mode, 
                      or using use-when, or using xsi:type, or creating elements in the schema or fn namespace,
                      or duplicates -->
                <!-- <skip file="{.}"/>-->
              </xsl:when>
              <xsl:otherwise>
                <xsl:message>Validating <xsl:value-of select="."/></xsl:message>
                <xsl:variable name="copy">
            	     <xsl:copy-of select="$doc" validation="strict"/>
                </xsl:variable>
                <xsl:message>Validated <xsl:value-of select="."/></xsl:message>
                <ok file="{.}" size="{count($copy//*)}"/>
                </xsl:otherwise>
              </xsl:choose>     
            <xsl:catch>
              <xsl:variable name="regex"> version=["']1\.1['"]</xsl:variable>
              <xsl:if use-when="not($supports_xml1.1)" test="not(unparsed-text(resolve-uri(., base-uri(..))) => matches($regex))">
                <xsl:message select="$err:code, $err:description"> ** Invalid ** </xsl:message>
                <invalid code="{$err:code}" file="{.}" message="{$err:description}"/>
              </xsl:if>
              <xsl:if use-when="$supports_xml1.1" test="true()">
                <xsl:message select="$err:code, $err:description"> ** Invalid ** </xsl:message>
                <invalid code="{$err:code}" file="{.}" message="{$err:description}"/>
              </xsl:if>
            </xsl:catch>
          </xsl:try>
        </xsl:for-each>  
      </xsl:variable>
      <out>
        <xsl:copy-of select="$results[self::invalid]"/>
      </out>
  </xsl:template>

</xsl:stylesheet>