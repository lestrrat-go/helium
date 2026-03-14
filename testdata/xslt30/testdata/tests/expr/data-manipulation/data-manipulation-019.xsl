<?xml version="1.0"?> 

 <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#format-number?>
  <!-- Purpose: Test of "format-number" using a parameter as 
       the number to be formatted and a variable for the
       format pattern. -->

<xsl:param name = "param1">1234.78</xsl:param>
<xsl:variable name = "variable1">#,###.00</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select = "format-number($param1,$variable1)"/>
 </out>
</xsl:template>

</xsl:stylesheet>
