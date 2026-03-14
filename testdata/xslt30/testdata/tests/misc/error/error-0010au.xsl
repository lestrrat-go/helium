<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- BUG: 5.4.1/008 -->
<!-- DESCRIPTION: No error is reported if xsl:text or xsl:value-of is used at the top level of the
    stylesheet. -->
    <?spec xslt#errors?><?error XTSE0010?>

   <xsl:text>This is not allowed!</xsl:text>
 
</xsl:stylesheet>
