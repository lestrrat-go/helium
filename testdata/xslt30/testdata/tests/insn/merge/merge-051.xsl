<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- xsl:merge with two homogenous files and validation by type -->
    <xsl:output method="xml" indent="no"/>
    <xsl:import-schema schema-location="books.xsd"/>
    
    <xsl:accumulator name="item-counter" as="xs:integer" initial-value="0">
        <xsl:accumulator-rule match="ITEM" select="$value + 1"/>
    </xsl:accumulator>
    
        <xsl:template name="main">
        <books>
            <xsl:merge>
                <xsl:merge-source for-each-source="'books1.xml', 'books2.xml'"
                    streamable="yes"
                    use-accumulators="item-counter"
                	select="BOOKLIST/BOOKS/ITEM"
                	type="BOOKLIST-type">
                        <xsl:merge-key select="accumulator-before('item-counter')"/>
                </xsl:merge-source>
                <xsl:merge-action>
                   <xsl:choose>
                     <xsl:when test="current-merge-group() instance of schema-element(ITEM)*">
                       <ok/>
                     </xsl:when>
                     <xsl:otherwise>
                       <bad><xsl:copy-of select="current-merge-group()"/></bad>
                     </xsl:otherwise>
                   </xsl:choose>   
                </xsl:merge-action>
            </xsl:merge>
        </books>
        
    </xsl:template>
</xsl:stylesheet>