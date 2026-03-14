<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-floor?>
  <!-- Purpose: Test of 'floor' function with a borderline 
       negative number.  The argument is relatively far from 
       the expected result. -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="floor(-1.0001)"/>
 </out>
</xsl:template>
</xsl:stylesheet>