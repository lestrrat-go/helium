<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    <xsl:param name="three" select="3" as="xs:integer"/>
    
    <!-- Streaming subsequence#2: grounded operand -->
    
    <xsl:template name="r-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="subsequence(data(/BOOKLIST/BOOKS/ITEM/PRICE), $three)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: grounded operand -->
    
    <xsl:template name="r-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="subsequence(copy-of(/BOOKLIST/BOOKS/ITEM/PRICE), 3)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: grounded atomic operand -->
    
    <xsl:template name="r-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/ITEM/DIMENSIONS ! subsequence(tokenize(., ' '), $three)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: striding operand, inspection usage -->
    
    <xsl:template name="r-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="count(subsequence(/BOOKLIST/BOOKS/ITEM/PRICE, 3))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: striding operand, absorption usage -->
    
    <xsl:template name="r-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(sum(subsequence(/BOOKLIST/BOOKS/ITEM/PRICE, 3)), 2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: striding operand, transmission usage -->
    
    <xsl:template name="r-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="subsequence(/BOOKLIST/BOOKS/ITEM/PRICE, $three)[position() mod 2 = 0]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: striding operand, focus-setting usage -->
    
    <xsl:template name="r-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="subsequence(/BOOKLIST/BOOKS/ITEM/PRICE, 3) ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: striding operand, focus-controlled usage -->
    
    <!-- Saxon 26 Nov 2013. Failing. A for-each that returns streamed nodes
         can't currently be handled by a ComposingWatch. Need to generalize
         TransmissionAdjunct -->
    
    <xsl:template name="r-014" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM[1] ! subsequence(*, 3)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: striding operand, focus-setting usage -->
    
    <xsl:template name="r-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="subsequence(/BOOKLIST/BOOKS/ITEM/PRICE, $three)">
            <xsl:value-of select=".+1 || ' '"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: striding operand, focus-controlled usage -->
    
    <xsl:template name="r-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM[1]">
            <xsl:copy-of select="subsequence(*, 3)"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>    
    
    <!-- Streaming subsequence#2: striding operand, focus-setting usage for inspection action -->
    
    <xsl:template name="r-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="subsequence(/BOOKLIST/BOOKS/ITEM[1]/*, 3) ! contains(name(), 'E')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: striding operand, apply-templates usage -->
    
    <xsl:template name="r-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:apply-templates select="subsequence(/BOOKLIST/BOOKS/ITEM[1]/*, $three)" mode="r-018-mode"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:mode name="r-018-mode" streamable="yes" on-no-match="deep-skip"/>
    <xsl:template match="ITEM/*" mode="r-018-mode"><xsl:value-of select="."/></xsl:template>
    
    <!-- Streaming subsequence#2: crawling operand, inspection usage -->
    
    <xsl:template name="r-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count(subsequence(//PRICE, 3))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Streaming subsequence#2: crawling operand, absorption usage -->
    
    <xsl:template name="r-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count(subsequence(/BOOKLIST/BOOKS/ITEM[1]//text(), 3))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: crawling operand, transmission usage -->
    
    <xsl:template name="r-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="subsequence(/BOOKLIST/BOOKS/ITEM[1]//text(), $three)[position() lt 4]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: crawling operand, focus-setting usage -->
    
    <xsl:template name="r-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="subsequence(//PRICE/text(), 3) ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: climbing operand, inspection usage -->
    
    <xsl:template name="r-030" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="subsequence(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*, 3) ! name()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: climbing operand, inspection usage, removal of duplicates -->
    
    <xsl:template name="r-031" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="subsequence(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor::*, $three) ! name()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: climbing operand, absorption usage -->
    
    <xsl:template name="r-032" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="subsequence(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*/@*, 3)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: climbing operand, absorption usage -->
    
    <xsl:template name="r-033" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="subsequence(/BOOKLIST/BOOKS/ITEM/ancestor-or-self::*/@CAT, 3)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-034" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="subsequence(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor-or-self::*/@CAT, $three)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming subsequence#2: climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-035" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="subsequence(//PRICE/ancestor-or-self::*/@*, 3)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>               
    
</xsl:stylesheet>