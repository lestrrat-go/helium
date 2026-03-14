<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="3.0">
    <!-- Purpose: xsl:merge-source that uses a path expression that selects more than one node in an 
    attribute value template  -->

    <xsl:variable name="doc1" select="doc('log-file-3.xml')"/>

    <xsl:template match="/">
        <results>

            <xsl:for-each select="1 to 2">
                <xsl:variable name="pos" select="position()"/>
                <result>
                    <xsl:merge>
                        <xsl:merge-source
                            select="if($pos lt 2) then $doc1//logs/events[@order='ascending']/event else $doc1//logs/events[@order='descending']/event">
                            <xsl:merge-key select="xs:dateTime(@timestamp)"
                                order="{if($pos lt 2) then 'ascending' else 'descending'}"/>
                        </xsl:merge-source>

                        <xsl:merge-action>
                            <xsl:copy-of select="current-merge-group()"/>
                        </xsl:merge-action>
                    </xsl:merge>
                </result>
            </xsl:for-each>
        </results>

    </xsl:template>
</xsl:stylesheet>
