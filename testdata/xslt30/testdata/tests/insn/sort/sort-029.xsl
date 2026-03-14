<?xml version="1.0"?> 

<?spec xslt#sorting?>
<?dependency saxon?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: Error: invalid lang attribute (dynamic detection)  -->

<xsl:param name="lang">de</xsl:param>

    
<xsl:output indent="yes" encoding="iso-8859-1"/>

<xsl:template match="/">
   <out lang="default-order">
    <xsl:for-each select="//alpha">
      <xsl:sort select="." data-type="text" lang="'{$lang}'"/>
      <xsl:value-of select="."/>
    </xsl:for-each>
   </out>
</xsl:template>
</xsl:stylesheet>
