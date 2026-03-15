<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- inspection access to current() in higher-order operand position -->
    
    <xsl:template name="c-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="head(/BOOKLIST/BOOKS/ITEM)">
            <xsl:copy-of select="*[namespace-uri(.) = namespace-uri(current())]"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- inspection access to current() in higher-order operand position -->
    
    <xsl:template name="c-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="head(/BOOKLIST/BOOKS/ITEM)">
            <xsl:copy-of select="*[.. is current()]"/>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- trivial use of current() -->
    
    <xsl:template name="c-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="head(/BOOKLIST/BOOKS/ITEM)">
            <xsl:copy-of select="current()"/>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- current() within a path expression -->
    
    <xsl:template name="c-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="head(/BOOKLIST/BOOKS/ITEM)">
            <xsl:copy-of select="current()/PRICE"/>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- current() applied to grounded items-->
    
    <xsl:template name="c-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="1 to 5">
            <a><xsl:value-of select="(20 to 21)[. gt current()]"/></a>
          </xsl:for-each>  
          <b><xsl:value-of select="count(//*)"/></b>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- current() with ancestor axis -->
    
    <xsl:template name="c-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
            <xsl:copy-of select="ancestor-or-self::*[name(.) eq name(current())]/local-name()"/>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
                
    
</xsl:stylesheet>