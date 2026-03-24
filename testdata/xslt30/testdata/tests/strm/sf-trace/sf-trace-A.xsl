<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- Streaming trace(): grounded operand -->
    
    <xsl:template name="r-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="trace(data(/BOOKLIST/BOOKS/ITEM/PRICE), 'r-001')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): grounded operand -->
    
    <xsl:template name="r-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="trace(copy-of(/BOOKLIST/BOOKS/ITEM/PRICE), 'r-002')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): grounded atomic operand -->
    
    <xsl:template name="r-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/ITEM/DIMENSIONS ! trace(tokenize(., ' '), 'r-003')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): striding operand, inspection usage -->
    
    <xsl:template name="r-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="count(trace(/BOOKLIST/BOOKS/ITEM/PRICE, 'r-010'))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): striding operand, absorption usage -->
    
    <xsl:template name="r-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(trace(/BOOKLIST/BOOKS/ITEM/PRICE, 'r-011'))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): striding operand, transmission usage -->
    
    <xsl:template name="r-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="trace(/BOOKLIST/BOOKS/ITEM/PRICE, 'label')[position() mod 2 = 0]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): striding operand, focus-setting usage -->
    
    <xsl:template name="r-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="trace(/BOOKLIST/BOOKS/ITEM/PRICE, 'r-013') ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): striding operand, focus-controlled usage -->
    
    <!-- Saxon 26 Nov 2013. Failing. A for-each that returns streamed nodes
         can't currently be handled by a ComposingWatch. Need to generalize
         TransmissionAdjunct -->
    
    <xsl:template name="r-014" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM[1] ! trace(*, 'r-014')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): striding operand, focus-setting usage -->
    
    <xsl:template name="r-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="trace(/BOOKLIST/BOOKS/ITEM/PRICE, 'r-015')">
            <xsl:value-of select=".+1 || ' '"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): striding operand, focus-controlled usage -->
    
    <xsl:template name="r-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM[1]">
            <xsl:copy-of select="trace(*, 'r-016')"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>    
    
    <!-- Streaming trace(): striding operand, focus-setting usage for inspection action -->
    
    <xsl:template name="r-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="trace(/BOOKLIST/BOOKS/ITEM[1]/*, 'r-017') ! contains(name(), 'E')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): striding operand, apply-templates usage -->
    
    <xsl:template name="r-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:apply-templates select="trace(/BOOKLIST/BOOKS/ITEM[1]/*, 'r-018')" mode="r-018-mode"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:mode name="r-018-mode" streamable="yes" on-no-match="deep-skip"/>
    <xsl:template match="ITEM/*" mode="r-018-mode"><xsl:value-of select="."/></xsl:template>
    
    <!-- Streaming trace(): crawling operand, inspection usage -->
    
    <xsl:template name="r-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count(trace(//PRICE, 'r-020'))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Streaming trace(): crawling operand, absorption usage -->
    
    <xsl:template name="r-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count(trace(/BOOKLIST/BOOKS/ITEM[1]//text(), 'r-021'))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): crawling operand, transmission usage -->
    
    <xsl:template name="r-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="trace(/BOOKLIST/BOOKS/ITEM[1]//text(), 'r-022')[position() lt 4]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): crawling operand, focus-setting usage -->
    
    <xsl:template name="r-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="trace(//PRICE/text(), 'r-023') ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): climbing operand, inspection usage -->
    
    <xsl:template name="r-030" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="trace(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*, 'r-030') ! name()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): climbing operand, inspection usage, removal of duplicates -->
    
    <xsl:template name="r-031" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="trace(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor::*, 'r-031') ! name()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): climbing operand, absorption usage -->
    
    <xsl:template name="r-032" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="trace(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*/@*, 'r-032')" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): climbing operand, absorption usage -->
    
    <xsl:template name="r-033" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="trace(/BOOKLIST/BOOKS/ITEM/ancestor-or-self::*/@CAT, 'r-033')" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-034" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="trace(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor-or-self::*/@CAT, 'r-034')" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming trace(): climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-035" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="trace(//PRICE/ancestor-or-self::*/@*, 'r-035')" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>               
    
</xsl:stylesheet>