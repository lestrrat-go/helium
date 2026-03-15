<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#initiating?>
<!-- PURPOSE:   Test use of an initial mode matching a template with mode="#all" (see config file)  -->
<!-- Following the resolution of http://www.w3.org/Bugs/Public/show_bug.cgi?id=3690 this test should raise an error
 (XTDE0045)  -->
<xsl:output indent="yes"/>

<xsl:template match="doc" mode="#all">
    <bozo/>
</xsl:template>
 
</xsl:stylesheet>
