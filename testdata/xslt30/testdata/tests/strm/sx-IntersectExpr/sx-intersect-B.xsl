<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:variable name="insertion" as="node()*">
      <a>A</a>
      <b>B</b>
      <xsl:text>InsertionText</xsl:text>
      <xsl:attribute name="Alpha" select="'InsertionAttribute'"/>
    </xsl:variable>
    
    <xsl:variable name="numeric-insertion" as="element()*">
      <a>11</a>
      <b>22</b>
    </xsl:variable>
    
    <!-- intersect expression (streaming arg 2): grounded operand -->
    
    <xsl:template name="r-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="($insertion intersect  copy-of(/BOOKLIST/BOOKS/ITEM/PRICE))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): striding operand, inspection usage -->
    
    <xsl:template name="r-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="count(($insertion intersect  /BOOKLIST/BOOKS/ITEM/PRICE))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): striding operand, absorption usage -->
    
    <xsl:template name="r-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(($numeric-insertion intersect /BOOKLIST/BOOKS/ITEM/PRICE))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): striding operand, transmission usage -->
    
    <xsl:template name="r-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="($insertion intersect  /BOOKLIST/BOOKS/ITEM/PRICE)[position() mod 2 = 0]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): striding operand, focus-setting usage -->
    
    <xsl:template name="r-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="($numeric-insertion intersect /BOOKLIST/BOOKS/ITEM/PRICE) ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): striding operand, focus-controlled usage -->
    
    <xsl:template name="r-014" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM[1] ! ($insertion intersect  *)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): striding operand, focus-setting usage -->
    
    <xsl:template name="r-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="($numeric-insertion intersect /BOOKLIST/BOOKS/ITEM/PRICE)">
            <xsl:value-of select=".+1 || ' '"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): striding operand, focus-controlled usage -->
    
    <xsl:template name="r-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM[1]">
            <xsl:copy-of select="($insertion intersect *)"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>    
    
    <!-- intersect expression (streaming arg 2): striding operand, focus-setting usage for inspection action -->
    
    <xsl:template name="r-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="($insertion intersect /BOOKLIST/BOOKS/ITEM[1]/*) ! contains(name(), 'E')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): striding operand, apply-templates usage -->
    
    <xsl:template name="r-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:apply-templates select="($insertion intersect /BOOKLIST/BOOKS/ITEM[1]/*)" mode="r-018-mode"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:mode name="r-018-mode" streamable="yes" on-no-match="deep-skip"/>
    <xsl:template match="ITEM/*" mode="r-018-mode"><xsl:value-of select="."/></xsl:template>
    
    <!-- intersect expression (streaming arg 2): crawling operand, inspection usage -->
    
    <xsl:template name="r-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count(($insertion intersect //PRICE))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- intersect expression (streaming arg 2): crawling operand, absorption usage -->
    
    <xsl:template name="r-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count(($insertion intersect /BOOKLIST/BOOKS/ITEM[1]//text()))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): crawling operand, transmission usage -->
    
    <xsl:template name="r-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="($insertion intersect /BOOKLIST/BOOKS/ITEM[1]//text())[position() lt 4]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): crawling operand, focus-setting usage -->
    
    <xsl:template name="r-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="($numeric-insertion intersect //PRICE/text()) ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): climbing operand, inspection usage -->
    
    <xsl:template name="r-030" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="($insertion intersect /BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*) ! name()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): climbing operand, inspection usage, removal of duplicates -->
    
    <xsl:template name="r-031" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="($insertion intersect  /BOOKLIST/BOOKS/ITEM/PRICE/ancestor::*) ! name()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): climbing operand, absorption usage -->
    
    <xsl:template name="r-032" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="($insertion intersect /BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*/@*)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): climbing operand, absorption usage -->
    
    <xsl:template name="r-033" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="($insertion intersect /BOOKLIST/BOOKS/ITEM/ancestor-or-self::*/@CAT)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-034" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="($insertion intersect /BOOKLIST/BOOKS/ITEM/PRICE/ancestor-or-self::*/@CAT)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- intersect expression (streaming arg 2): climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-035" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="($insertion intersect //PRICE/ancestor-or-self::*/@*)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming intersect operator: striding operand, transmission usage  -->
    
    <xsl:template name="r-036" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/*/CATEGORIES/CATEGORY">
            <v><xsl:value-of select="@DESC intersect @CODE"/></v>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>  
    
    <!-- Streaming intersect operator: two climbing/motionless operands both comprising streamed nodes -->
    
    <xsl:template name="r-037" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="outermost(//WEIGHT)">
            <v><xsl:value-of select="ancestor-or-self::*/@CAT intersect ../@*"/></v>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming intersect operator: a motionless and a striding operand both comprising streamed nodes -->
    
    <xsl:template name="r-038" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/*/BOOKS/ITEM[1]">
            <xsl:value-of select="(. intersect *) ! local-name()"/>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>   
    
    <!-- Streaming intersect operator: a motionless and a striding operand both comprising streamed nodes -->
    
    <xsl:template name="r-039" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/*/BOOKS/ITEM[1]">
            <xsl:value-of select="(. | .) ! local-name()"/>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>  
  
  <!-- Streaming intersect operator: a motionless and a striding operand both comprising streamed nodes -->
  
  <xsl:template name="r-040" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="/*/BOOKS/ITEM[1]">
          <xsl:value-of select="(. intersect descendant-or-self::*) ! local-name()"/>
        </xsl:for-each>
      </out>
    </xsl:source-document>
  </xsl:template>  
    
                  
    
</xsl:stylesheet>