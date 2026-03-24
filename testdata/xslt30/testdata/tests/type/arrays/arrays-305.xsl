<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:math="http://www.w3.org/2005/xpath-functions/math"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:array="http://www.w3.org/2005/xpath-functions/array"
  exclude-result-prefixes="xs math map array">
  
  <!-- do a random sequence of operations to construct an array, then
    do some retrievals -->
  
  <xsl:template match="/" name="xsl:initial-template">
    <out>
      <xsl:variable name="a" select="array:join((array{1 to 20}, array{101 to 120})) 
        => array:subarray(2, 36)
        => array:reverse()
        => array:put(15, 1000)
        => array:remove(19)
        => array:tail()
        => array:insert-before(12, 2000)"/>
      <a><xsl:sequence select="$a"/></a>
      <b><xsl:sequence select="array:size($a)"/></b>
      <c><xsl:sequence select="array:get($a, 5)"/></c>
      <d><xsl:sequence select="array:subarray($a, 6, 30)"/></d>
    </out>
  </xsl:template>
</xsl:transform>
