<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- Streaming outermost(): grounded operand -->
    
    <xsl:template name="r-001" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="data(outermost(/BOOKLIST/BOOKS/ITEM/PRICE))"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): grounded operand -->
    
    <xsl:template name="r-002" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:copy-of select="outermost(copy-of(/BOOKLIST/BOOKS/ITEM/PRICE))"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): grounded atomic operand -->
    
    <xsl:template name="r-003" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="outermost(/BOOKLIST/BOOKS/ITEM/DIMENSIONS) ! tokenize(., ' ')"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): striding operand, inspection usage -->
    
    <xsl:template name="r-010" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="count(outermost(/BOOKLIST/BOOKS/ITEM/PRICE))"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): striding operand, absorption usage -->
    
    <xsl:template name="r-011" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="sum(outermost(/BOOKLIST/BOOKS/ITEM/PRICE))"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): striding operand, transmission usage -->
    
    <xsl:template name="r-012" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="outermost(/BOOKLIST/BOOKS/ITEM/PRICE)[position() mod 2 = 0]"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): striding operand, focus-setting usage -->
    
    <xsl:template name="r-013" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="outermost(/BOOKLIST/BOOKS/ITEM/PRICE) ! (.+1)"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): striding operand, focus-controlled usage -->
    
    <!-- Saxon 26 Nov 2013. Failing. A for-each that returns streamed nodes
         can't currently be handled by a ComposingWatch. Need to generalize
         TransmissionAdjunct -->
    
    <xsl:template name="r-014" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM[1] ! outermost(*)"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): striding operand, focus-setting usage -->
    
    <xsl:template name="r-015" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:for-each select="outermost(/BOOKLIST/BOOKS/ITEM/PRICE)">
            <xsl:value-of select=".+1 || ' '"/>
          </xsl:for-each>  
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): striding operand, focus-controlled usage -->
    
    <xsl:template name="r-016" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM[1]">
            <xsl:copy-of select="outermost(*)"/>
          </xsl:for-each>  
        </out>
      </xsl:for-each>
    </xsl:template>    
    
    <!-- Streaming outermost(): striding operand, focus-setting usage for inspection action -->
    
    <xsl:template name="r-017" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="outermost(/BOOKLIST/BOOKS/ITEM[1]/*) ! contains(name(), 'E')"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): striding operand, apply-templates usage -->
    
    <xsl:template name="r-018" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:apply-templates select="outermost(/BOOKLIST/BOOKS/ITEM[1]/*)" mode="r-018-mode"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <xsl:mode name="r-018-mode" streamable="yes" on-no-match="deep-skip"/>
    <xsl:template match="ITEM/*" mode="r-018-mode"><xsl:value-of select="."/></xsl:template>
    
    <!-- Streaming outermost(): crawling operand, inspection usage -->
    
    <xsl:template name="r-020" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:copy-of select="count(outermost(//PRICE))"/>
        </out>
      </xsl:for-each>
    </xsl:template> 
    
    <!-- Streaming outermost(): crawling operand, absorption usage -->
    
    <xsl:template name="r-021" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:copy-of select="count(outermost(/BOOKLIST/BOOKS/ITEM[1]//text()))"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): crawling operand, transmission usage -->
    
    <xsl:template name="r-022" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:copy-of select="outermost(/BOOKLIST/BOOKS/ITEM[1]//text())[position() lt 4]"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): crawling operand, focus-setting usage -->
    
    <xsl:template name="r-023" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:copy-of select="outermost(//PRICE/text()) ! (.+1)"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): climbing operand, inspection usage -->
    
    <xsl:template name="r-030" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:copy-of select="outermost(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*) ! name()"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): climbing operand, inspection usage, removal of duplicates -->
    
    <xsl:template name="r-031" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:copy-of select="outermost(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor::*) ! name()"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): climbing operand, absorption usage -->
    
    <xsl:template name="r-032" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="outermost(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*/@*)" separator="|"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): climbing operand, absorption usage -->
    
    <xsl:template name="r-033" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="outermost(/BOOKLIST/BOOKS/ITEM/ancestor-or-self::*/@CAT)" separator="|"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-034" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="outermost(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor-or-self::*/@CAT)" separator="|"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-035" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/books.xml')">
        <out>
          <xsl:value-of select="outermost(//PRICE/ancestor-or-self::*/@*)" separator="|"/>
        </out>
      </xsl:for-each>
    </xsl:template> 
    
    <!-- Streaming outermost(): recursive data -->
    
    <xsl:template name="r-100" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/recursive.xml')">
        <out>
          <xsl:value-of select="outermost(//section)/@id" separator="|"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <!-- Streaming outermost(): recursive data -->
    
    <xsl:template name="r-101" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/recursive.xml')">
        <out>
          <xsl:value-of select="outermost(//section/section)/@id" separator="|"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <xsl:template name="r-102" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/recursive.xml')">
        <out>
          <xsl:value-of select="outermost(//section/section)/head" separator="|"/>
        </out>
      </xsl:for-each>
    </xsl:template>
    
    <xsl:template name="r-103" use-when="$RUN">
      <xsl:for-each select="doc('../../strm/docs/recursive.xml')">
        <out>
          <xsl:value-of select="outermost(//section/section)/foot" separator="|"/>
        </out>
      </xsl:for-each>
    </xsl:template>               
    
</xsl:stylesheet>