<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/local-functions"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <!-- A simple preprocessor for XSLT stylesheets implementing use-when, static variables, and shadow attributes
       (within a single stylesheet module only) -->
  
  <xsl:mode streamable="yes" on-no-match="shallow-copy" use-accumulators="static-vars"/>
  
  <xsl:accumulator name="static-vars" as="map(xs:QName, item()*)" initial-value="map{}" streamable="yes">
     <xsl:accumulator-rule match="xsl:variable[@static=('yes', '1', 'true')]" phase="end">
       <xsl:variable name="select" as="item()*">
         <xsl:evaluate xpath="@select" with-params="accumulator-before('static-vars')"/>
       </xsl:variable>
       <xsl:sequence select="map:put($value, resolve-QName(@name, .), $select)"/>
     </xsl:accumulator-rule>
  </xsl:accumulator>
  
  <xsl:template match="xsl:*[@use-when] | (* except xsl:*)[@xsl:use-when]">
    <xsl:variable name="use" as="xs:boolean">
      <xsl:evaluate xpath="@*:use-when" with-params="accumulator-before('static-vars')"/>
    </xsl:variable>
    <xsl:if test="$use">
        <xsl:next-match/>
    </xsl:if>
  </xsl:template>
  
  <xsl:template match="xsl:*/@*[starts-with(local-name(), '_')][namespace-uri()='']">
    <xsl:variable name="params" select="../accumulator-before('static-vars')"/>
    <xsl:attribute name="{substring(local-name(), 2)}">
      <xsl:variable name="regex">\{([^{}]+)\}</xsl:variable>
      <!-- restriction: doesn't allow embedded expressions to contain curly braces -->
      <xsl:analyze-string select="." regex="{$regex}">
        <xsl:matching-substring>
          <xsl:evaluate xpath="regex-group(1)" with-params="$params"/>
        </xsl:matching-substring>
        <xsl:non-matching-substring>
          <xsl:value-of select="."/>
        </xsl:non-matching-substring>
      </xsl:analyze-string>
    </xsl:attribute>
  </xsl:template>
  
</xsl:stylesheet>
