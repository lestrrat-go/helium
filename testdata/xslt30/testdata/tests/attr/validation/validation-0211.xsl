<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns="http://www.w3.org/1999/xhtml"
    exclude-result-prefixes="xs"
    version="2.0">

    <!-- xsl:result-document validation: no href; invalid -->
    
    <xsl:output method="xhtml" indent="yes"/>
    
    <!-- import the schema for the target XHTML vocabulary -->

  <xsl:import-schema namespace="http://www.w3.org/1999/xhtml"
    schema-location="xhtml1-transitional.xsd"/>

    <xsl:param name="dir" select="'.'"/>

    <xsl:template match="/">
        <xsl:result-document validation="strict">
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
                            <xsl:value-of select="ancestor::IndividualRec/IndivName[1]"/>
                        </p>
                    </xsl:for-each>
                </xsl:for-each-group>
            </body>
            <xsl:if test="current-date() gt xs:date('1900-01-01')">
              <body>Two bodies is bad news</body>
            </xsl:if>
        </html>
    </xsl:template>

</xsl:transform>
