<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:param name="two" as="element()">
      <a>2</a>
    </xsl:param>
    
    
    <!-- Streaming arithmetic operator: consuming striding operand -->
    
    <xsl:template name="r-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM/PRICE)[1] + 2"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming arithmetic operator: consuming crawling operand -->
    
    <xsl:template name="r-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="outermost(//PRICE)[1] + $two"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming arithmetic operator: striding motionless operand -->
    
    <xsl:template name="r-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <out>
          <xsl:for-each select="//transaction">
            <a><xsl:value-of select="@value + $two"/></a>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming arithmetic operator: consuming crawling operand selecting empty sequence -->
    
    <xsl:template name="r-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(//PRICE)[82] + $two"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming arithmetic operator: two operands from streamed document -->
    
    <xsl:template name="r-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <out>
          <xsl:for-each select="//transaction">
            <a><xsl:value-of select="format-number(@value + string-length(@date), '0.00')"/></a>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming arithmetic operator: two operands from streamed document -->
    
    <xsl:template name="r-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="outermost(//PRICE)">
            <a><xsl:value-of select="format-number(. + string-length(../@CAT), '0.00')"/></a>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming arithmetic operator: complexity in non-streamed operand -->
    
    <xsl:template name="r-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="outermost(//PRICE)">
            <a><xsl:value-of select="format-number(. + (let $c := string(../@CAT) return if (string-length($c) gt 2) then string-length($c) else 0), '0.00')"/></a>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming arithmetic operator: complexity in streamed operand -->
    
    <xsl:template name="r-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="outermost(//PRICE)">
            <a><xsl:value-of select="(let $p := round(number(.)) return max(($p, $p - 1))) + $two"/></a>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>       
    
</xsl:stylesheet>