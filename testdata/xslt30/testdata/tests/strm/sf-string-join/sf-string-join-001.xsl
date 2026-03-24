<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local.functions/"
    exclude-result-prefixes="map xs f">
    
    <!-- Test of xsl:source-document calling string-join(), first argument consuming and striding  -->
    
    <xsl:template name="j-001">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="string-join(/BOOKLIST/BOOKS/ITEM/TITLE, ', ')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling string-join(), first argument motionless, second argument streamed  -->
    
    <xsl:template name="j-002">      
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="string-join((1 to 10)!string(), /BOOKLIST/BOOKS/ITEM[1]/@CAT)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling string-join(), first argument consuming and striding, second argument omitted  -->
    
    <xsl:template name="j-003">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="string-join(/BOOKLIST/BOOKS/ITEM/TITLE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    
    
</xsl:stylesheet>