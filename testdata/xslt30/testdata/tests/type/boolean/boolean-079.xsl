<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-logical-expressions?>
<!-- PURPOSE: Test 3-valued truth table for OR -->
<?same-as-1.0 no?>
<xsl:template match="/">
<out>
  true or true is <xsl:value-of select="true() or true()"/>
  true or false is <xsl:value-of select="true() or false()"/>  
  true or () is <xsl:value-of select="true() or ()"/>  
  false or true is <xsl:value-of select="false() or true()"/>  
  false or false is <xsl:value-of select="false() or false()"/>  
  false or () is <xsl:value-of select="false() or ()"/>  
  () or true is <xsl:value-of select="() or true()"/>  
  () or false is <xsl:value-of select="() or false()"/>  
  () or () is <xsl:value-of select="() or ()"/>    
</out>
</xsl:template>
</xsl:stylesheet>
