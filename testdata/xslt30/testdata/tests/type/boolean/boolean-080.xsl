<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- DOCUMENT: http://www.w3.org/TR/xpath -->
<!-- SECTION: 3.4 Booleans -->
<!-- PURPOSE: Test 3-valued truth table for AND with computed booleans -->
<?same-as-1.0 no?>
<?spec xpath#id-logical-expressions?>
<xsl:template match="/">
<out>
  true and true is <xsl:value-of select="count(//*) &gt; 3 and count(//*) &lt; 1000000"/>
  true and false is <xsl:value-of select="count(//*) &gt; 3 and count(//*) &gt; 1000000"/>  
  true and () is <xsl:value-of select="count(//*) &gt; 3 and //banana"/>  
  false and true is <xsl:value-of select="exists(//banana) and exists(.)"/>  
  false and false is <xsl:value-of select="exists(//banana) and exists(//apple)"/>  
  false and () is <xsl:value-of select="4=5 and //*[1000000]"/>  
  () and true is <xsl:value-of select="/fruit and 5 = 10 div 2"/>  
  () and false is <xsl:value-of select="/fruit and 2=3"/>  
  () and () is <xsl:value-of select="/apple and /pear"/>    
</out>
</xsl:template>
</xsl:stylesheet>
