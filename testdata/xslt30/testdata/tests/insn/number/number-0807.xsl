<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test formatting of a number larger than an integer  -->
  
  <xsl:template match="doc">
  <!--Results changed in 8.2.1, large numbers can now be formatted-->
    <out>
		<xsl:number value="1e100" format="[1]" />
    </out>
  </xsl:template>
 
</xsl:stylesheet>
