<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- Streaming if() expression: grounded operand -->
    
    <xsl:template name="r-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="if(exists(/BOOKLIST/BOOKS/ITEM/PRICE)) then 0 else 1"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: grounded operand -->
    
    <xsl:template name="r-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="if(copy-of(/BOOKLIST/BOOKS/ITEM/PRICE)) then 0 else 1"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: grounded atomic operand -->
    
    <xsl:template name="r-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/ITEM/DIMENSIONS ! (if(exists(tokenize(., ' '))) then 0 else 1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: striding operand, inspection usage -->
    
    <xsl:template name="r-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="count(if(/BOOKLIST/BOOKS/ITEM/PRICE) then 0 else 1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: striding operand, absorption usage -->
    
    <xsl:template name="r-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(if(/BOOKLIST/BOOKS/ITEM/PRICE) then 0 else 1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: striding operand, transmission usage -->
    
    <xsl:template name="r-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="if((/BOOKLIST/BOOKS/ITEM/PRICE)[position() mod 2 = 0]) then 0 else 1"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: striding operand, focus-setting usage -->
    
    <xsl:template name="r-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="if((/BOOKLIST/BOOKS/ITEM/PRICE) ! (.+1) = 5.95)  then 0 else 1"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: striding operand, focus-controlled usage -->
    
    <xsl:template name="r-014" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM[1] ! (if(*) then 0 else 1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: striding operand, focus-setting usage -->
    
    <xsl:template name="r-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="if(/BOOKLIST/BOOKS/ITEM/PRICE) then 0 else 1">
            <xsl:value-of select=".+1 || ' '"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: striding operand, focus-controlled usage -->
    
    <xsl:template name="r-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM[1]">
            <xsl:copy-of select="if(*) then 0 else 1"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>    
    
    <!-- Streaming if() expression: striding operand, focus-setting usage for inspection action -->
    
    <xsl:template name="r-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(if(/BOOKLIST/BOOKS/ITEM[1]/*)  then 0 else 1) ! contains(string(), 'E')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: striding operand, apply-templates usage -->
    
    <xsl:template name="r-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:apply-templates select="if(/BOOKLIST/BOOKS/ITEM[1]/*) then 0 else 1" mode="r-018-mode"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:mode name="r-018-mode" streamable="yes" on-no-match="deep-skip"/>
    <xsl:template match="ITEM/*" mode="r-018-mode"><xsl:value-of select="."/></xsl:template>
    <xsl:template match=".[. instance of xs:integer]" expand-text="yes" mode="r-018-mode">#{.}</xsl:template>
    
    <!-- Streaming if() expression: crawling operand, inspection usage -->
    
    <xsl:template name="r-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count(if(//PRICE) then 0 else 1)"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Streaming if() expression: crawling operand, absorption usage -->
    
    <xsl:template name="r-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count(if(/BOOKLIST/BOOKS/ITEM[1]//text()) then 0 else 1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: crawling operand, transmission usage -->
    
    <xsl:template name="r-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="if((/BOOKLIST/BOOKS/ITEM[1]//text())[position() lt 4])  then 0 else 1"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: crawling operand, focus-setting usage -->
    
    <xsl:template name="r-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(if(//PRICE/text())  then 0 else 1) ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: climbing operand, inspection usage -->
    
    <xsl:template name="r-030" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(if(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*)  then 0 else 1) ! string()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: climbing operand, inspection usage, removal of duplicates -->
    
    <xsl:template name="r-031" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(if(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor::*)  then 0 else 1) ! string()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: climbing operand, absorption usage -->
    
    <xsl:template name="r-032" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="if(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*/@*) then 0 else 1" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: climbing operand, absorption usage -->
    
    <xsl:template name="r-033" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="if(/BOOKLIST/BOOKS/ITEM/ancestor-or-self::*/@CAT) then 0 else 1" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-034" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="if(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor-or-self::*/@CAT) then 0 else 1" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming if() expression: climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-035" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="if(//PRICE/ancestor-or-self::*/@*) then 0 else 1" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>               
    
</xsl:stylesheet>