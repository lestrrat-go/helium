<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0"
    exclude-result-prefixes="#all">

    <xsl:variable name="regex" as="xs:string"
        select="'^(\p{Lu}{2}_\p{Lu}{3}_)?((\d{5})_)?(\d{5}).*$'"/>
    <xsl:variable name="basename" as="xs:string" select="'UV_STD_00000_39000_Test'"/>

    <xsl:template name="main">
        <out>
            <xsl:analyze-string select="$basename" regex="{$regex}">
                <xsl:matching-substring>
                    <xsl:value-of
                        select="
                            for $i in (1 to 4)
                            return
                                concat($i, ':', regex-group($i))"
                    />
                </xsl:matching-substring>
            </xsl:analyze-string>
        </out>
    </xsl:template>
</xsl:stylesheet>
