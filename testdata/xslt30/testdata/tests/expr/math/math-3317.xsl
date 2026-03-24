<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <!-- test BCM: arithmetic on empty sequence returns NaN -->

    <xsl:template match="/">
        <out>
            <a0 ans="{format-number(() + (), '#.##')}"/>
            <a0 ans="{format-number(() * 12, '#.##')}"/>
            <a0 ans="{format-number(10 div (), '#.##')}"/>
        </out>
    </xsl:template>
</xsl:stylesheet>
