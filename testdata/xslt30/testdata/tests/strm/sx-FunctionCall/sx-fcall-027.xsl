<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local.functions/"
    exclude-result-prefixes="map xs f">
    
    <!-- Test of xsl:stream calling a (higher-order) function -->
    <!-- Not streamable without some very smart analysis... -->
    
    <xsl:template name="main">
      <xsl:source-document streamable="yes" href="books.xml">
        <out>
          <xsl:value-of select="max(//ITEM/DIMENSIONS!f:dimensions(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:function name="f:dimensions" as="xs:decimal">
      <!-- return the product of the dimensions -->
      <xsl:param name="item" as="element(DIMENSIONS)"/>
      <xsl:sequence select="fold-left(xs:NMTOKENS($item)!xs:decimal(.), 1, function($x, $y) {$x*$y})"/>
    </xsl:function>
    
</xsl:stylesheet>