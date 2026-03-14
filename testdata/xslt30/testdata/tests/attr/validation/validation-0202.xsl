<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns="http://www.w3.org/1999/xhtml"
    version="2.0">


    <xsl:import href="validation-0202-person.xsl"/>
    
    <xsl:output method="xhtml" indent="yes"/>

    <xsl:param name="dir" select="'.'"/>

    <xsl:template match="/">
        <xsl:for-each select="*/IndividualRec">
            <xsl:result-document href="{$dir}/validation-0202-out/{@Id}.xhtml" validation="strict"
                encoding="iso-8859-1">
                <xsl:apply-templates select="."/>
            </xsl:result-document>
        </xsl:for-each>
        <xsl:result-document href="{$dir}/validation-0202-out/index.xhtml" validation="strict"
            encoding="iso-8859-1">
            <xsl:call-template name="make-index"/>
        </xsl:result-document>
    </xsl:template>

    <xsl:template name="make-index">
        <html>
            <head>
                <title>Index of names</title>
            </head>
            <body>
                <h1>Index of names</h1>
                <xsl:for-each-group select="/*/IndividualRec/IndivName/NamePart[@Level=1]"
                    group-by=".">
                    <xsl:sort select="current-grouping-key()"/>
                    <h2>
                        <xsl:value-of select="current-grouping-key()"/>
                    </h2>
                    <xsl:for-each select="current-group()">
                        <p>
                            <xsl:apply-templates select="ancestor::IndividualRec/IndivName[1]"
                                mode="link"/>
                        </p>
                    </xsl:for-each>
                </xsl:for-each-group>
            </body>
        </html>
    </xsl:template>

</xsl:transform>
