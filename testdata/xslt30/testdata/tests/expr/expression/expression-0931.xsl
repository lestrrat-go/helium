<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
exclude-result-prefixes="xsi"
>

<!-- test nilled() function in an untyped environment (always false) -->
<?spec fo#func-nilled?>

<xsl:variable name="e" as="element()"><e xsi:nil="true"/></xsl:variable>

<xsl:variable name="j">
  <a thing="thong"><!--comment--><?pi?>text</a>
</xsl:variable>

<xsl:template match="/">
<z>
  <d><xsl:value-of select="nilled($j)"/></d>
  <e><xsl:value-of select="nilled($j/a)"/></e>  
  <a><xsl:value-of select="nilled($j/a/thing)"/></a> 
  <p><xsl:value-of select="nilled($j/a/processing-instruction())"/></p>   
  <c><xsl:value-of select="nilled($j/a/comment())"/></c>   
  <t><xsl:value-of select="nilled($j/a/text())"/></t>   
  <n><xsl:value-of select="nilled($j/a/namespace::xml)"/></n>
  <s><xsl:value-of select="nilled(())"/></s>
  <x><xsl:value-of select="nilled($e)"/></x>  
</z>       
</xsl:template>


</xsl:stylesheet>
