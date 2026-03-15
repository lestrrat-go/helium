<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
    expand-text="yes" version="3.0">
    
    <xsl:strip-space elements="*"/>

    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:template name="main">
        <root>
            <subjects>
                <xsl:merge>
                    <xsl:merge-source name="books" for-each-source="'merge-091b.xml'" streamable="yes" select="books/book" >
                        <xsl:merge-key select="subject"/>
                    </xsl:merge-source>
                    <xsl:merge-source name="casts" for-each-source="'merge-091a.xml'" 
                        select="root/list/cast" streamable="yes">
                        <xsl:merge-key select="@subject"/>
                    </xsl:merge-source>
                    <xsl:merge-action>
                        <subject name="{current-merge-key()}">
                            <xsl:apply-templates select="current-merge-group('casts'), current-merge-group('books')"/>
                        </subject>
                    </xsl:merge-action>
                </xsl:merge>
            </subjects>
        </root>
    </xsl:template>
    
    <xsl:template match="book">
        <media type="{local-name()}">
            <xsl:apply-templates select="* except subject"/>
        </media>
    </xsl:template>
    
    <xsl:template match="cast">
        <media type="{local-name()}">
            <title>{@title}</title>
        </media>
    </xsl:template>
    
</xsl:stylesheet>