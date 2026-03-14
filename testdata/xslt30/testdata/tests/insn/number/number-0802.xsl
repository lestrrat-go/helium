<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test non-english numbering (relies on Numberer_de being present) -->
  <xsl:template match="doc">
    <out>
        <xsl:number value="3" format="w" lang="de" letter-value="traditional"/>;
        <xsl:number value="13" format="w" lang="de" letter-value="traditional"/>;
        <xsl:number value="13" format="a" lang="de"/>;
        <xsl:number value="819" format="a"  lang="de"/>;
        <xsl:number value="1999" format="A"  lang="de"/>;
        <xsl:number value="100000" format="A"  lang="de"/>;
    </out>
  </xsl:template>
 
</xsl:stylesheet>
