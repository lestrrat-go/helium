<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="#all" version="3.0"
    xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:template name="xsl:initial-template">
        <xsl:try>
            <xsl:message terminate="yes">
                <test>Take me to your leader<xsl:comment>REALLY</xsl:comment></test>
            </xsl:message>
            <xsl:catch>
                <err>
                    <code>
                        <xsl:value-of select="$err:code" />
                    </code>
                    <description>
                        <xsl:value-of select="$err:description" />
                    </description>
                    <value>
                        <xsl:sequence select="$err:value" />
                    </value>
                </err>
            </xsl:catch>
        </xsl:try>
    </xsl:template>
</xsl:stylesheet>