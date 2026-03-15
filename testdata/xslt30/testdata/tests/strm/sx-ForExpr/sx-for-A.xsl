<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>

    <!-- "For" expression with consuming and grounded "in" operand.
        Use two references to the range variable to avoid inlining. -->
    
    <xsl:template name="r-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="for $x in /BOOKLIST/BOOKS/ITEM/DIMENSIONS/string() return sum(tokenize($x, ' ')!xs:decimal(.)) + string-length($x)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:template name="r-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="for $x in /BOOKLIST/BOOKS/ITEM/DIMENSIONS/data() return sum(tokenize($x, ' ')!xs:decimal(.)) + string-length($x)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:template name="r-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="for $x in /BOOKLIST/BOOKS/ITEM/DIMENSIONS/copy-of() return sum(tokenize($x, ' ')!xs:decimal(.)) + string-length($x)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:template name="r-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="
            for $x in /BOOKLIST/BOOKS/ITEM/DIMENSIONS/snapshot() 
            return sum(tokenize($x, ' ')!xs:decimal(.)) + count($x/ancestor::node())"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:template name="r-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="
            for $x in /BOOKLIST/BOOKS/ITEM/DIMENSIONS/ancestor-or-self::*/@*/node-name() 
            return ('Q{'||namespace-uri-from-QName($x)||'}'||local-name-from-QName($x))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
         
    
</xsl:stylesheet>