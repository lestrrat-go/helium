<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}"/>
    
    <xsl:output method="html" indent="yes"/>
    
    <xsl:template match="/">
        <html>
            <head>
                <title>Population grouped by country</title>
            </head>
            <body>
                <xsl:apply-templates/>
            </body>
        </html>
    </xsl:template>
    
    <xsl:template match="cities">
        <table>
            <tr>
                <th>Position</th>
                <th>Country</th>
                <th>City List</th>
                <th>Population</th>
            </tr>
            <xsl:fork>
                <xsl:for-each-group select="city" group-by="@country">
                    <tr>
                        <td>
                            <xsl:value-of select="position()"/>
                        </td>
                        <td>
                            <xsl:value-of select="current-grouping-key()"/>
                        </td>
                        <xsl:variable name="data" 
                            select="map { 'cities' : current-group()/@name/string(), 'pop-sum' : sum(current-group()/@pop) }"/>
                        <td>
                            <xsl:value-of select="$data?cities" separator=", "/>
                        </td>
                        <td>
                            <xsl:value-of select="$data?pop-sum"/>
                        </td>
                    </tr>
                </xsl:for-each-group>
            </xsl:fork>
        </table>
    </xsl:template>
    
</xsl:stylesheet>