<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:for-each select="1 to 10">
                <xsl:if test="position() mod 2" expand-text="yes">
                    <!-- separator has no influence, the sequence is combined into a text node before it kicks in -->
                    <xsl:value-of separator="">{'foo-', position()}</xsl:value-of>
                </xsl:if>
            </xsl:for-each>
        </out>
    </xsl:template>

</xsl:stylesheet>
