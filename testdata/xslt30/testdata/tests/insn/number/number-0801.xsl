<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test alpha numbering -->
  <xsl:template match="doc">
    <out>
        <xsl:number value="13" format="a"/>;
        <xsl:number value="819" format="a"/>;
        <xsl:number value="1999" format="A"/>;
        <xsl:number value="100000" format="A"/>;
    </out>
  </xsl:template>
 
</xsl:stylesheet>
