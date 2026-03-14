<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-number?>
  <!-- Purpose: Test of 'number' function with a negative
       borderline value as its argument. -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="number('-1.9999')"/>
 </out>
</xsl:template>
</xsl:stylesheet>