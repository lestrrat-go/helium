<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:f="http://f.com/" exclude-result-prefixes="f xs"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">

    <!-- test round-half-to-even on decimals -->

    <xsl:param name="big" select="1234567896123456789612345678961234567896.1234567896"/>
    <xsl:param name="big2" select="1234567896500000000000000000000000000000.1234567896"/>
    <xsl:param name="big3" select="1234567897500000000000000000000000000000.1234567896"/>
    <xsl:param name="small" select="0.0000000000000000000000000123456789012345678901"/>

    <xsl:template match="/">
        <out>
            <a0 ans="{round-half-to-even($big, 40)}"/>
            <aa ans="{round-half-to-even($big, 4)}"/>
            <ab ans="{round-half-to-even($big, 1)}"/>
            <ac ans="{round-half-to-even($big, 0)}"/>
            <ad ans="{round-half-to-even($big, -1)}"/>
            <ae ans="{round-half-to-even($big, -10)}"/>
            <af ans="{round-half-to-even($big, -30)}"/>
            <af2 ans="{round-half-to-even($big2, -30)}"/>
            <af3 ans="{round-half-to-even($big3, -30)}"/>
            <ag ans="{round-half-to-even($big, -100)}"/>
            <b0 ans="{round-half-to-even($small, 40)}"/>
            <ba ans="{round-half-to-even($small, 4)}"/>
            <bb ans="{round-half-to-even($small, 1)}"/>
            <bc ans="{round-half-to-even($small, 0)}"/>
            <bd ans="{round-half-to-even($small, -1)}"/>
            <be ans="{round-half-to-even($small, -10)}"/>
            <bf ans="{round-half-to-even($small, -30)}"/>
            <bg ans="{round-half-to-even($small, -100)}"/>
        </out>
    </xsl:template>
</xsl:stylesheet>
