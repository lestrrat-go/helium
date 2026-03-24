<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" expand-text="yes" version="3.0">
    <xsl:template match="/" name="main">
        <out>
            <xsl:variable name="x">Time now: {current-dateTime()}</xsl:variable>
            <time now="{$x}" then="{$x}"/>
        </out>
    </xsl:template>
</xsl:stylesheet>
