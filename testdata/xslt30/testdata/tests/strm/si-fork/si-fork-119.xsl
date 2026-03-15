<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="#all" expand-text="yes"
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}"/>
    
    <xsl:output method="adaptive"/>
    
    <xsl:template match=".[. instance of map(xs:string, element()?)]">
        <xsl:param name="current-item" as="element()" select="?current-item"/>
        <xsl:param name="previous-item" as="element()?" select="?previous-item"/>
        <xsl:param name="next-item" as="element()?" select="?next-item"/>
        <xsl:for-each select="$current-item">
            
            <xsl:result-document href="item_{@id}.html" method="html" indent="yes" version="5.0">
                <html>
                    <head>
                        <title>Item {@id}</title>
                    </head>
                    <body>
                        <h1>Item {@id}</h1>
                        <nav>
                            <xsl:if test="$previous-item">
                                <a href="item_{$previous-item/@id}.html">prev</a>
                            </xsl:if>
                            <xsl:if test="$next-item">
                                <a href="item_{$next-item/@id}.html">next</a>
                            </xsl:if>
                        </nav>
                    </body>
                </html>
            </xsl:result-document>
        </xsl:for-each>
    </xsl:template>
    
    <xsl:template match="/">
        <html>
            <head>
                <title>Testing xsl:iterate</title>
            </head>
            <body>
                <h1>Testing xsl:iterate</h1>
                <section>
                    <h2>Results</h2>
                    <xsl:apply-templates/>
                </section>
            </body>
        </html>
    </xsl:template>
    
    <xsl:template match="items">
        <ul>
            <xsl:fork>
                <xsl:sequence>
                    <xsl:apply-templates select="item"/>
                </xsl:sequence>
                <xsl:sequence>
                    <xsl:iterate select="item">
                        <xsl:param name="previous-item" as="element(item)?" select="()"/>
                        <xsl:param name="current-item" as="element(item)*" select="()"/>
                        
                        <xsl:on-completion>
                            <xsl:apply-templates select="map { 'current-item' : $current-item, 'previous-item' : $previous-item }"/>
                        </xsl:on-completion>
                        
                        <xsl:variable name="this-item" select="copy-of()"/>
                        
                        <xsl:if test="position() gt 1">
                            <xsl:apply-templates select="map { 'current-item' : $current-item, 'previous-item' : $previous-item, 'next-item' : $this-item }"/>
                        </xsl:if>
                        
                        <xsl:next-iteration>
                            <xsl:with-param name="current-item" select="$this-item"/>
                            <xsl:with-param name="previous-item" select="$current-item"/>
                        </xsl:next-iteration>
                        
                    </xsl:iterate>
                </xsl:sequence>
            </xsl:fork>
        </ul>
    </xsl:template>
    
    <xsl:template match="item">
        <li>Item {@id}, position: {position()}, attribute pos: {@pos}</li>
    </xsl:template>
    
</xsl:stylesheet>