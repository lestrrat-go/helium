<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-string?>
  <!-- Purpose: Test of 'string()' function with a variable reference 
       as the argument. -->

<xsl:variable name="variable1">String From Variable</xsl:variable>
<xsl:template match="doc">
 <out>
   <xsl:value-of select="string($variable1)"/>
 </out>
</xsl:template>
</xsl:stylesheet>