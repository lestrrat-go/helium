<?spec xslt#grouping?>
<xsl:transform version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <!-- Demonstrate group-by using a numeric key (mixing different numeric types) -->

    <xsl:template match="/">
        <o>
            <xsl:for-each-group select="//city"
                group-by="if (@country='england') then number(@pop) else 7">
                <group pop="{if (@country='england') then number(@pop) else 7}">
                    <xsl:copy-of select="current-group()"/>
                </group>
            </xsl:for-each-group>
        </o>
    </xsl:template>

</xsl:transform>
