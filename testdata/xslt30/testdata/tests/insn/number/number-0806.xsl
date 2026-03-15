<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test level="any" when there are no matches  -->
  <xsl:template match="doc">
    <out>
		<xsl:number level="any" count="nothing" format="[1]" />;
    </out>
  </xsl:template>
 
</xsl:stylesheet>
