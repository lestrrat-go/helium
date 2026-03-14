<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  exclude-result-prefixes="xs map"
  version="3.0">
  
  <xsl:mode streamable="yes" use-accumulators="type-count"/>
  
  <xsl:output method="text"/>
  
  <xsl:accumulator name="type-count" as="map(xs:string, xs:integer)" initial-value="map{}" streamable="yes">
    <xsl:accumulator-rule match="product">
      <xsl:sequence select="if (map:contains($value, @type))
        then map:put($value, string(@type), $value(@type) + 1)
        else map:put($value, string(@type), 1)"/>
    </xsl:accumulator-rule>
  </xsl:accumulator>
  
  <xsl:template match="/">
    <xsl:for-each select="*/*">
      <xsl:message select="node-name(.)"/>
    </xsl:for-each>
    <xsl:value-of select="serialize(accumulator-after('type-count'), map{ 'method' : 'adaptive'})"/>
  </xsl:template>
  
</xsl:stylesheet>