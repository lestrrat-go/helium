<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-general-comparisons?>
<!-- PURPOSE: Test of "=" operator with mixed types -->
<!-- Modified from bool56 which does string comparisons at XPath 2.0 -->
<?same-as-1.0 no?>
<xsl:template match="/">
<out>
  <xsl:variable name="set1" select="//set1/value"/>
  <xsl:variable name="set2" select="(1, '2')"/>
  <xsl:value-of select="$set1 = $set2"/> should be true
  <xsl:value-of select="$set2 != $set1"/> should be true
  <xsl:value-of select="$set2 &gt; $set1"/> should be false  
  <xsl:value-of select="$set2 &lt; $set1"/> should be true    
</out>
</xsl:template>
</xsl:stylesheet>
