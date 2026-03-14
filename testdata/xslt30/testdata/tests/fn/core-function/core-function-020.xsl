<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-concat?>
  <!-- Purpose: Test of 'concat()' function with a variable reference 
       as the argument.  Only the second argument of the function is
       build by referencing a variable. -->

<xsl:variable name="variable1">From </xsl:variable>
<xsl:template match="doc">
 <out>
   <xsl:value-of select="concat('String ',$variable1,'Variable')"/>
 </out>
</xsl:template>
</xsl:stylesheet>