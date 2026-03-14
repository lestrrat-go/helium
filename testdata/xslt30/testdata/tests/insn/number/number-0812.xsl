<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:output encoding="iso-8859-1"/>

  <!-- PURPOSE: test numbering in German words -->
  <xsl:template match="doc">
    <out>
        <n3><xsl:number value="3" format="w" lang="de"/></n3>
        <n10><xsl:number value="10" format="w" lang="de"/></n10>
        <n13><xsl:number value="13" format="w" lang="de"/></n13>
        <n20><xsl:number value="20" format="w"  lang="de"/></n20>
        <n100><xsl:number value="100" format="W" lang="de"/></n100>
        <n115><xsl:number value="115" format="W" lang="de"/></n115>
        <n134><xsl:number value="134" format="W" lang="de"/></n134>
        <n200><xsl:number value="200" format="w" lang="de"/></n200>
        <n201><xsl:number value="201" format="w" lang="de"/></n201>
        <n210><xsl:number value="210" format="w" lang="de"/></n210>
        <n230><xsl:number value="230" format="w"  lang="de"/></n230>
        <n1000><xsl:number value="1000" format="W" lang="de"/></n1000>
        <n1005><xsl:number value="1005" format="W" lang="de"/></n1005>
        <n2134816><xsl:number value="2134816" format="Ww" lang="de"/></n2134816>       
    </out>
  </xsl:template>
 
</xsl:stylesheet>
