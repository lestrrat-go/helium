<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
  <!-- Non-streamable fork/for-each-group: population is crawling -->
  
  <xsl:template name="xsl:initial-template">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
          <xsl:fork>
            <xsl:for-each-group select="//ITEM" group-by="@CAT">
              <in a="{count(current-group())}"/>
            </xsl:for-each-group>
          </xsl:fork>
      </out>  
    </xsl:source-document>
  </xsl:template>
  
  
</xsl:transform>  