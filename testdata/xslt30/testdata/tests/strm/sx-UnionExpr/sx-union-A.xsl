<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:variable name="insertion" as="element()*">
      <a>A</a>
      <b>B</b>
    </xsl:variable>
    
    <xsl:variable name="numeric-insertion" as="element()*">
      <a>11</a>
      <b>22</b>
    </xsl:variable>
  
  <xsl:variable name="extraItem" as="element(ITEM)">
    <ITEM CAT="H">
      <TITLE>Ulysses</TITLE>
      <AUTHOR>James Joyce</AUTHOR>
      <PUBLISHER>HarperCollins</PUBLISHER>
      <PUB-DATE>1935-06-02</PUB-DATE>
      <LANGUAGE>English</LANGUAGE>
      <PRICE>18.90</PRICE>
      <QUANTITY>235</QUANTITY>
      <ISBN>0186701805</ISBN>
      <PAGES>830</PAGES>
      <DIMENSIONS UNIT="in">7.2 5.6 1.2</DIMENSIONS>
      <WEIGHT UNIT="oz">11.0</WEIGHT>
    </ITEM>
  </xsl:variable>
    
    <!-- Streaming union operator: grounded operand -->
    
    <xsl:template name="r-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(copy-of(/BOOKLIST/BOOKS/ITEM/PRICE) union  $insertion)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: striding operand, inspection usage -->
    
    <xsl:template name="r-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="count((/BOOKLIST/BOOKS/ITEM/PRICE union  $insertion))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: striding operand, absorption usage -->
    
    <xsl:template name="r-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum((/BOOKLIST/BOOKS/ITEM/PRICE union  $numeric-insertion))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: striding operand, transmission usage -->
    
    <xsl:template name="r-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM/PRICE union  $insertion)[position() mod 2 = 0]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: striding operand, focus-setting usage -->
    
    <xsl:template name="r-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM/PRICE union  $numeric-insertion) ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: striding operand, focus-controlled usage -->
    
    <xsl:template name="r-014" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM[1] ! (* union  $insertion)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: striding operand, focus-setting usage -->
    
    <xsl:template name="r-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="(/BOOKLIST/BOOKS/ITEM/PRICE union  $numeric-insertion)">
            <xsl:value-of select=".+1 || ' '"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: striding operand, focus-controlled usage -->
    
    <xsl:template name="r-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM[1]">
            <xsl:copy-of select="(* union $insertion)"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>    
    
    <!-- Streaming union operator: striding operand, focus-setting usage for inspection action -->
    
    <xsl:template name="r-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM[1]/* union  $insertion) ! contains(name(), 'E')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: striding operand, apply-templates usage -->
    
    <xsl:template name="r-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:apply-templates select="(/BOOKLIST/BOOKS/ITEM[1]/* union $insertion)" mode="r-018-mode"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:mode name="r-018-mode" streamable="yes" on-no-match="deep-skip"/>
    <xsl:template match="ITEM/*" mode="r-018-mode"><xsl:value-of select="."/></xsl:template>
    
    <!-- Streaming union operator: crawling operand, inspection usage -->
    
    <xsl:template name="r-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count((//PRICE union  $insertion))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Streaming union operator: crawling operand, absorption usage -->
    
    <xsl:template name="r-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="count((/BOOKLIST/BOOKS/ITEM[1]//text() union  $insertion))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: crawling operand, transmission usage -->
    
    <xsl:template name="r-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(/BOOKLIST/BOOKS/ITEM[1]//text() union  $insertion)[position() lt 4]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: crawling operand, focus-setting usage -->
    
    <xsl:template name="r-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(//PRICE/text() union  $numeric-insertion) ! (.+1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: climbing operand, inspection usage -->
    
    <xsl:template name="r-030" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::* union  $insertion) ! name()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: climbing operand, inspection usage, removal of duplicates -->
    
    <xsl:template name="r-031" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor::* union  $insertion) ! name()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: climbing operand, absorption usage -->
    
    <xsl:template name="r-032" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM[1]/PRICE/ancestor::*/@* union  $insertion)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: climbing operand, absorption usage -->
    
    <xsl:template name="r-033" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM/ancestor-or-self::*/@CAT union $insertion)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-034" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM/PRICE/ancestor-or-self::*/@CAT union  $insertion)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming union operator: climbing operand, absorption usage, removal of duplicates -->
    
    <xsl:template name="r-035" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(//PRICE/ancestor-or-self::*/@* union  $insertion)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>  
    
    <!-- Streaming union operator: striding operand, transmission usage (bug 27668) -->
    
    <xsl:template name="r-036" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="/*/CATEGORIES/CATEGORY">
            <v><xsl:value-of select="@CODE union @DESC"/></v>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>
  
  <!-- Mixed striding and grounded sequence.  -->
  
  <xsl:template name="r-040" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:copy-of select="($extraItem | /BOOKLIST/BOOKS/ITEM)!PRICE"/>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Mixed striding and grounded sequence.  -->
  
  <xsl:template name="r-041" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:copy-of select="($extraItem | /BOOKLIST/BOOKS/ITEM) / PRICE"/>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Mixed striding and grounded sequence.  -->
  
  <xsl:template name="r-042" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="($extraItem | /BOOKLIST/BOOKS/ITEM)!PRICE"/>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Mixed striding and grounded sequence.  -->
  
  <xsl:template name="r-043" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="($extraItem | /BOOKLIST/BOOKS/ITEM) / PRICE"/>
      </out>
    </xsl:source-document>
  </xsl:template> 
    
</xsl:stylesheet>