<?xml version="1.0"?> 

<?spec xslt#sorting?>
<?dependency saxon?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: Error: invalid lang attribute (static detection)  -->



    
<xsl:output indent="yes" encoding="iso-8859-1"/>

<xsl:template match="/">
   <out lang="default-order">
    <xsl:for-each select="//item">
      <xsl:sort select="." data-type="text" lang="'de'"/>
      <xsl:value-of select="."/>
    </xsl:for-each>
   </out>
</xsl:template>
</xsl:stylesheet>
