<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-floor?>
  <!-- Purpose: Test of 'floor' function with a borderline 
       negative number.  The argument is very close to 
       expected result. -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="floor(-1.99999)"/>
 </out>
</xsl:template>
</xsl:stylesheet>