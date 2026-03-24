
<xsl:stylesheet
   version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:one="http://one.uri/"
   xmlns:def="http://default.uri/">
   
<xsl:variable name="doc" select="/def:doc"/>
<xsl:variable name="chap" select="/def:doc/chap"/>
<xsl:variable name="data1" select="/def:doc/chap/data-one"/>
<xsl:variable name="data2" select="/def:doc/chap/data-two"/>

<xsl:template match="/">
<out>
  <xsl:variable name="res0" select="resolve-QName($chap/@att-one, $doc)"/>
  {<xsl:value-of select="namespace-uri-from-QName($res0)"/>}<xsl:value-of select="local-name-from-QName($res0)"/><xsl:text>#xa;</xsl:text>
  <xsl:variable name="res1" select="resolve-QName($chap/@att-one, $chap)"/>
  {<xsl:value-of select="namespace-uri-from-QName($res1)"/>}<xsl:value-of select="local-name-from-QName($res1)"/><xsl:text>#xa;</xsl:text>
  <xsl:variable name="res2" select="resolve-QName(string($data1), $data1)"/> 
  {<xsl:value-of select="namespace-uri-from-QName($res2)"/>}<xsl:value-of select="local-name-from-QName($res2)"/><xsl:text>#xa;</xsl:text>
  <xsl:variable name="res3" select="resolve-QName(string($data2), $data2)"/>
  {<xsl:value-of select="namespace-uri-from-QName($res3)"/>}<xsl:value-of select="local-name-from-QName($res3)"/><xsl:text>#xa;</xsl:text>
</out>
</xsl:template>
</xsl:stylesheet>
