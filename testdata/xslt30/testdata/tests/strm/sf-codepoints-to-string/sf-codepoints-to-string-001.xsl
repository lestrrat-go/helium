<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local.functions/"
    exclude-result-prefixes="map xs f">
    
    <!-- Test of xsl:source-document calling codepoints-to-string()  -->
    
    <xsl:template name="c-001">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="codepoints-to-string(outermost(//TITLE)/(30+string-length(.)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling codepoints-to-string() - error case -->
    
    <xsl:template name="c-002">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="codepoints-to-string(outermost(//TITLE)/(string-length(.) - 60))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling codepoints-to-string() - error case, recovered -->
    
    <xsl:template name="c-003">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="codepoints-to-string(outermost(//TITLE)/(string-length(.) - 60))"/>
            <xsl:catch errors="*:FOCH0001" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling codepoints-to-string(), empty case  -->
    
    <xsl:template name="c-004">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="codepoints-to-string(outermost(//TITTLE-TATTLE)/(30+string-length(.)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    
</xsl:stylesheet>