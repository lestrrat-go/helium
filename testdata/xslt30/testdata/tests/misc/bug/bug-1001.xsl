<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <?spec xpath#combining_seq?>
    <!-- BUG: Saxon 5.3.1/005 -->
    <!-- DESCRIPTION: A union expression whose operands are individual attribute nodes of the same element
    is evaluated incorrectly. For example count(@a | @b) returns 1. The attributes are wrongly
    regarded as duplicate nodes. -->


    <xsl:template match="/">
        <out>
            <xsl:for-each select="//*">
                <element name="{name()}">
                    <union>
                        <xsl:apply-templates select="@a | @b | @c">
                          <xsl:sort select="local-name()"/>
                        </xsl:apply-templates>
                    </union>
                    <list>
                        <xsl:apply-templates select="@*">
                          <xsl:sort select="local-name()"/>
                        </xsl:apply-templates>
                    </list>
                </element>
            </xsl:for-each>
        </out>
    </xsl:template>

    <xsl:template match="@*">
        <attribute name="{name()}" value="{.}" last="{last()}"/>
    </xsl:template>

</xsl:stylesheet>
