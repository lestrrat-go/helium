<?xml version="1.0"?> 

<!-- mode-1426 -->
<!-- Michael Kay / mod. by ABr-->
<!-- on-no-match=fail, failure case, with streaming; the error is caught, and reported back with a motionless construct --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <xsl:mode on-no-match="fail" streamable="yes" />

    <xsl:template match="/">
        <xsl:apply-templates />
    </xsl:template>

    <xsl:template match="*">
        <xsl:copy>
            <xsl:apply-templates />
        </xsl:copy>
    </xsl:template>

    <xsl:template match="text()[not(parent::v)]">
        <xsl:value-of select="." />
    </xsl:template>

    <xsl:template match="v">
        <xsl:copy>
            <xsl:try>
                <xsl:apply-templates />
                <xsl:catch errors="err:XTDE0555" xmlns:err="http://www.w3.org/2005/xqt-errors"
                    exclude-result-prefixes="err">
                    <c>error XTDE0555 caught</c>
                </xsl:catch>
            </xsl:try>
        </xsl:copy>
    </xsl:template>


</xsl:stylesheet>
