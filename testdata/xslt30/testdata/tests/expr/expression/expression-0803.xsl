<?xml version="1.0"?>

<xsl:stylesheet  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="2.0" >

<!-- test identity of nodes in a result tree fragment -->
<?spec xslt#generate-id?>

<xsl:variable name="fredRTF" >
    <Fred/>
</xsl:variable>
<xsl:variable name="fred" select="$fredRTF/node()" />

<xsl:variable name="billRTF" >
    <Bill/>
</xsl:variable>
<xsl:variable name="bill" select="$billRTF/node()" />

<xsl:template match="/">
    <!--<xsl:message>Modified 2002-08-08 to remove dependency on document number: expect "false"</xsl:message>-->
    <xsl:variable name="ids">
        <source><xsl:value-of select="generate-id(*)"/></source>
        <fred><xsl:value-of select="generate-id($fred)"/></fred>
        <bill><xsl:value-of select="generate-id($bill)"/></bill>
    </xsl:variable>
    <result comparison="{$ids/fred = $ids/bill}"/>
</xsl:template>

</xsl:stylesheet>
