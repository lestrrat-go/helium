<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <xsl:mode name="stream" streamable="yes" on-no-match="shallow-copy"/>
    <xsl:output method="xml" indent="yes"/>
    <!--Define Global Variables by Streaming Input XML -->
    <xsl:param name="fileHref" select="'si-group-039.xml'"/>
    <xsl:variable name="outDir" select="substring-before($fileHref, tokenize($fileHref,'/')[last()])"/>
    <xsl:variable name="OutFileName" select="substring-before($fileHref,'.xml') || '-Formatted.xml'"/>
    <xsl:template match="/" name="main">
        <xsl:result-document href="si-group-039.out">
            <xsl:source-document streamable="yes" href="{$fileHref}">
                <xsl:apply-templates mode="stream"/>
            </xsl:source-document>
        </xsl:result-document>
    </xsl:template>
    <xsl:template match="Groups" mode="stream">
        <xsl:copy>
            <xsl:for-each-group select="Group/copy-of(.)" group-by="Group_Detail/GroupIdentifier">
                <xsl:variable name="GroupDetail" select="Group_Detail/copy-of(.)"/>
                <xsl:variable name="groupId" select="$GroupDetail/GroupIdentifier"/>
                <Formatted-Group groupId="{$groupId}">
                    <xsl:copy-of select="current-group()/specifics"/>
                </Formatted-Group>
            </xsl:for-each-group>
        </xsl:copy>
    </xsl:template>
</xsl:transform>

