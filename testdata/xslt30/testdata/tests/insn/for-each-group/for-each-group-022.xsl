<?spec xslt#grouping?>
<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- Use case for group-adjacent to structure a DT/DD style glossary -->
  
  <xsl:template match="/">
    <o><xsl:apply-templates/></o>
  </xsl:template>  
  
<xsl:template match="DL">
  <xsl:variable name="groups">
    <xsl:for-each-group select="*" group-adjacent="name()">
      <xsl:choose>
        <xsl:when test="self::DT">
            <DTS><xsl:copy-of select="current-group()"/></DTS>
        </xsl:when>      
        <xsl:when test="self::DD">
            <DDS><xsl:copy-of select="current-group()"/></DDS>
        </xsl:when>
        <xsl:otherwise/> 
      </xsl:choose>   
    </xsl:for-each-group>
  </xsl:variable>
  <UL>	
    <xsl:for-each-group select="$groups/*" group-starting-with="DTS">
      <LI>
        <B>
          <xsl:for-each select="current-group()/DT">
            <xsl:value-of select="."/>
            <xsl:if test="position()!=last()">, </xsl:if>
          </xsl:for-each>
        </B>
        <xsl:choose>
          <xsl:when test="count(current-group()/DD) = 1">
            <I><xsl:value-of select="current-group()/DD"/></I>  
          </xsl:when>
          <xsl:otherwise>
            <OL>
              <xsl:for-each select="current-group()/DD">
                <LI><xsl:value-of select="."/></LI>
              </xsl:for-each>
            </OL>
          </xsl:otherwise>  
        </xsl:choose>
      </LI>
    </xsl:for-each-group>
  </UL>
</xsl:template>
	
</xsl:transform>