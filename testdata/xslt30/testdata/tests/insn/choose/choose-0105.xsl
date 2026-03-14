<xsl:stylesheet 
    xmlns:xsl    = "http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs    = "http://www.w3.org/2001/XMLSchema" 
    xmlns:svg    = "http://www.w3.org/2000/svg" 
    exclude-result-prefixes="xs" 
    xmlns:w="http://www.werum.com" 
    version="2.0">
    
    <xsl:variable name="BOX_WIDTH"    as="xs:double"    select="0"/>
    <xsl:variable name="BOX_HEIGHT"    as="xs:double"    select="0"/>
    
    <xsl:variable name="CIRCLE_R"    as="xs:double"    select="0"/>
    <xsl:variable name="FONT_SIZE"    as="xs:double"    select="0"/>
    
    <xsl:template match="/">
        <out>
            <xsl:call-template name="drawItem">
                <xsl:with-param name="itemType">StartStep</xsl:with-param>
                <xsl:with-param name="itemX" select="10"/>
                <xsl:with-param name="itemY" select="20"/>
            </xsl:call-template>
        </out>
    </xsl:template>
    
    <xsl:template name="drawItem">
        <xsl:param as="xs:double" name="itemX"/>
        <xsl:param as="xs:double" name="itemY"/>
        <xsl:param as="xs:string" name="itemType" />
        
        <xsl:choose>
            
            <xsl:when test="$itemType='StartStep' or $itemType='EndStep'">
                <svg:circle  
                    r="{$CIRCLE_R}" 
                    cx="{$itemX + $CIRCLE_R }" 
                    cy="{$itemY + $CIRCLE_R }"/>
            </xsl:when>
            <xsl:when test="$itemType='Splitting'">
                <svg:line  
                    x1="{$itemX}" 
                    y1="{$itemY}" 
                    x2="{$itemX + $BOX_WIDTH }" 
                    y2="{$itemY}" 
                    style="stroke:rgb(0,0,0);stroke-width:5" 
                />
            </xsl:when>
            <xsl:otherwise>
                <xsl:choose>
                    <xsl:when test="$itemType='Synchronisation'">
                        <svg:line  
                            x1="{$itemX}" 
                            y1="{$itemY }" 
                            x2="{$itemX + $BOX_WIDTH div 2}" 
                            y2="{$itemY + $BOX_HEIGHT div 2}" 
                            style="stroke:rgb(0,0,0);stroke-width:5" 
                        />
                    </xsl:when>
                    <xsl:otherwise>
                        <svg:rect 
                            x        = "{$itemX}" 
                            y        = "{$itemY}" 
                            width    = "{$BOX_WIDTH}" 
                            height    = "{$BOX_HEIGHT}" 
                            stroke    = "black" 
                            fill    = "none"/>
                    </xsl:otherwise>
                </xsl:choose>
            </xsl:otherwise>
        </xsl:choose>
    </xsl:template>        
</xsl:stylesheet>