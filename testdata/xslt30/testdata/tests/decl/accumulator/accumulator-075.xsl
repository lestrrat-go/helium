<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="#all" expand-text="yes"
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}" use-accumulators="#all"/>
    
    <xsl:accumulator name="last-id" as="xs:string?" initial-value="()" _streamable="{$STREAMABLE}" saxon:trace="yes" xmlns:saxon="http://saxon.sf.net/">
        <xsl:accumulator-rule match="item" phase="end" select="string(@id)"/>
    </xsl:accumulator>
   
    
    <xsl:template match="/">
        <html>
            <head>
                <title>Testing accumulator with streaming and result-document threading</title>
            </head>
            <body>
                <h1>Testing accumulator with streaming and result-document threading</h1>
                <section>
                    <h2>Results</h2>
                    <xsl:apply-templates/>
                </section>
            </body>
        </html>
    </xsl:template>
    
    <xsl:template match="items">
        <xsl:result-document href="item-results.xml">
            <html>
                <head>
                    <title>Items</title>
                </head>
                <body>
                    <ul>
                        <xsl:apply-templates select="item"/>
                    </ul>
                </body>
            </html>
        </xsl:result-document>
    </xsl:template>
    
    <xsl:template match="item">
        <li id="{@id}" pos="{position()}" attPos="{@pos}" accum="{accumulator-before('last-id')}"/>
        <xsl:result-document href="item_{@id}.xml">
            <Item id="{@id}" accum="{accumulator-before('last-id')}"/>
        </xsl:result-document>
    </xsl:template>
    
</xsl:stylesheet>