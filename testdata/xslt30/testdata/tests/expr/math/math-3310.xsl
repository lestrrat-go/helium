<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:template match="/">

        <!-- test value comparisons on NaN -->

        <xsl:variable name="a" select="2"/>
        <xsl:variable name="n" select="number(dummy)"/>
        <xsl:variable name="s" select="(2,3,4)"/>
        <xsl:variable name="t" select="($n, 4)"/>
        <z>
            <x>
                <f>
                    <xsl:value-of select="$a eq $n"/>
                </f>
                <t>
                    <xsl:value-of select="$a ne $n"/>
                </t>
                <f>
                    <xsl:value-of select="$a lt $n"/>
                </f>
                <f>
                    <xsl:value-of select="$a le $n"/>
                </f>
                <f>
                    <xsl:value-of select="$a gt $n"/>
                </f>
                <f>
                    <xsl:value-of select="$a ge $n"/>
                </f>
            </x>
            <x>
                <f>
                    <xsl:value-of select="$n eq $a"/>
                </f>
                <t>
                    <xsl:value-of select="$n ne $a"/>
                </t>
                <f>
                    <xsl:value-of select="$n lt $a"/>
                </f>
                <f>
                    <xsl:value-of select="$n le $a"/>
                </f>
                <f>
                    <xsl:value-of select="$n gt $a"/>
                </f>
                <f>
                    <xsl:value-of select="$n ge $a"/>
                </f>
            </x>
            <x>
                <t>
                    <xsl:value-of select="$s = $t"/>
                </t>
                <t>
                    <xsl:value-of select="$s != $t"/>
                </t>
                <t>
                    <xsl:value-of select="$s &lt; $t"/>
                </t>
                <t>
                    <xsl:value-of select="$s &lt;= $t"/>
                </t>
                <f>
                    <xsl:value-of select="$s &gt; $t"/>
                </f>
                <t>
                    <xsl:value-of select="$s &gt;= $t"/>
                </t>
            </x>
            <x>
                <t>
                    <xsl:value-of select="$t = $s"/>
                </t>
                <t>
                    <xsl:value-of select="$t != $s"/>
                </t>
                <f>
                    <xsl:value-of select="$t &lt; $s"/>
                </f>
                <t>
                    <xsl:value-of select="$t &lt;= $s"/>
                </t>
                <t>
                    <xsl:value-of select="$t &gt; $s"/>
                </t>
                <t>
                    <xsl:value-of select="$t &gt;= $s"/>
                </t>
            </x>


        </z>
    </xsl:template>
</xsl:stylesheet>
