<?xml version="1.0" ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<?spec ser#cdata-section-elements?>
<!-- BUG: Saxon 5.5/002 -->
<!-- DESCRIPTION: When the cdata-section-elements attribute of xsl:output is used, disable-output-escaping
    on xsl:text or xsl:value-of does not always work correctly. -->


<xsl:output method="xml" omit-xml-declaration="yes"
cdata-section-elements="testatr"/>

<xsl:variable name="s">&lt;output-escaping&gt;?
</xsl:variable>

<xsl:template match="/">
    <test>
        <testatr>
                This is cdata text
        </testatr>
        <testatr>
        <xsl:text disable-output-escaping="yes">&lt;a&gt;cdata+disable&lt;/a&gt;</xsl:text>
        </testatr>
        <testatr>
        cdata <a>without</a> disable
        </testatr>
    </test>
</xsl:template>

</xsl:stylesheet >

