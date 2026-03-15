<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:math="http://www.w3.org/2005/xpath-functions/math"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:array="http://www.w3.org/2005/xpath-functions/array"
  exclude-result-prefixes="xs math map array">
  
  <xsl:output method="adaptive" indent="yes"/>
  
  <xsl:param name="array1" as="array(xs:integer)" select="[1, 2, 3, 4]"/>
  <xsl:param name="array2" as="array(xs:integer)"
    select="array:join(($array1, $array1))"/>
  <xsl:param name="array3" as="array(xs:integer)" select="[1, 2, 3, 4,
    1, 2, 3, 4]"/>
  <xsl:param name="array4" as="array(*)"
    select="array:subarray($array2, 3, 4)"/>
  
  <xsl:template match="/" name="xsl:initial-template">
    <out>
      <xsl:sequence select="$array1, $array2, $array3"/>
      <xsl:sequence select="$array4"/>
    </out>
  </xsl:template>
</xsl:transform>
