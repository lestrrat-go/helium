<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="x xs">

<!-- stylesheet functions with declared types of argument -->

<xsl:function name="x:product" as="xs:integer" >
    <xsl:param name="acc" as="xs:integer?"/>
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence 
         select="if ($n = 1) 
                 then $acc
                 else x:product($acc*$n, $n - 1)" />
</xsl:function>

<xsl:function name="x:factorial" as="xs:integer">
  <xsl:param name="n" as="xs:integer"/>
  <xsl:sequence  select="x:product(1, $n)"/>
</xsl:function>

  <xsl:template match="/">
      <out><xsl:value-of select="x:factorial(5)"/></out>
  </xsl:template>


</xsl:stylesheet>
