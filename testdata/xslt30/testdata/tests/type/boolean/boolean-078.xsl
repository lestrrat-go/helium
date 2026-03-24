<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-logical-expressions?>
<!-- PURPOSE: Test 3-valued truth table for AND -->
<?same-as-1.0 no?>
<xsl:template match="/">
<out>
  true and true is <xsl:value-of select="true() and true()"/>
  true and false is <xsl:value-of select="true() and false()"/>  
  true and () is <xsl:value-of select="true() and ()"/>  
  false and true is <xsl:value-of select="false() and true()"/>  
  false and false is <xsl:value-of select="false() and false()"/>  
  false and () is <xsl:value-of select="false() and ()"/>  
  () and true is <xsl:value-of select="() and true()"/>  
  () and false is <xsl:value-of select="() and false()"/>  
  () and () is <xsl:value-of select="() and ()"/>    
</out>
</xsl:template>
</xsl:stylesheet>
