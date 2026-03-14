<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#global-variables?>
<?error XTDE0050?>

<!-- PURPOSE:  It's an error if no value is supplied for a required parameter, even if
               the parameter is never referenced -->

<xsl:output indent="yes"/>

<xsl:param name="limbo" required="yes"/>

<xsl:template match="doc" mode="inimode">
    <bozo/>
</xsl:template>
 
</xsl:stylesheet>
