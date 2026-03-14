<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-round?>
  <!-- Purpose: Test of 'round' function with negative infinity
       as the argument (use expression that yields -infinity). -->
       <?same-as-1.0 no?>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="round(-2.3 div 0e0)"/>
 </out>
</xsl:template>
</xsl:stylesheet>