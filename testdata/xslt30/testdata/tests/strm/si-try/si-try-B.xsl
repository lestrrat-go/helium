<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>

    
    <!-- Test of xsl:source-document/xsl:try with avg() -->
    
    <xsl:template name="s-103" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg(./BOOKLIST/BOOKS/ITEM/PRICE), 2)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), filtered with a motionless predicate -->
    
    <xsl:template name="s-104" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg(./BOOKLIST/BOOKS/ITEM[@CAT='P']/PRICE)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), filtered with a positional predicate -->
    
    <xsl:template name="s-105" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg(./BOOKLIST/BOOKS/ITEM[position() lt 4]/PRICE), 2)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), filtered with a positional predicate -->
    
    <xsl:template name="s-106" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg(./BOOKLIST/BOOKS/*[position() lt 4]/PRICE), 2)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), filtered with a positional predicate -->
    
    <xsl:template name="s-107" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg(./BOOKLIST/BOOKS/*:ITEM[position() lt 4]/PRICE), 2)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), filtered using data() to make it streamable -->
    
    <xsl:template name="s-108" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg(./BOOKLIST/BOOKS/ITEM/PAGES/data()[. &lt; 1000][. &gt; 0]))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), filtered using text() to make it streamable -->
    
    <xsl:template name="s-109" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg(./BOOKLIST/BOOKS/ITEM/PAGES/text()[. &lt; 1000][. &gt; 0]))">
          	<xsl:catch errors="err:FORG0001" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), filtered using number() to make it streamable -->
    
    <xsl:template name="s-110" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg(./BOOKLIST/BOOKS/ITEM/PAGES/number()[. &lt; 1000][. &gt; 0]))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), selecting nothing and returning empty result -->
    
    <xsl:template name="s-111" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg(ITEM/PAGES)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg() selecting both streamed nodes and literals -->
    
    <xsl:template name="s-112" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg((./BOOKLIST/BOOKS/ITEM/PAGES/number(), 31, 32))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg() selecting both streamed nodes and literals while also filtering -->
    
    <xsl:template name="s-113" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg((tail(./BOOKLIST/BOOKS/ITEM/PAGES)/number(), 31, 32)), 2)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>   
    
    <!-- Test of xsl:source-document/xsl:try with sum of a computed value -->
    
    <xsl:template name="s-115" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="format-number(avg(for $d in data(outermost(//DIMENSIONS))
                                                  return let $x := tokenize($d, '\s')!number() 
                                                  return $x[1]*$x[2]*$x[3]), '99.999')"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with sum of a computed value -->
    
    <xsl:template name="s-116" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try select="xs:integer(round(avg(account/transaction/(@value*2))))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg(), filtered using copy-of() to make it streamable -->
    
    <xsl:template name="s-117" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="round(avg(./BOOKLIST/BOOKS/ITEM/PAGES/copy-of()[. &lt; 1000][. &gt; 0]))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg() applied to an attribute -->
    
    <xsl:template name="s-118" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try select="round(avg(account/transaction/@value))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg() over attribute values, filtered -->
    
    <xsl:template name="s-119" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try select="round(avg(account/transaction/@value[xs:decimal(.) gt 0]))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
  <!-- Test of xsl:source-document/xsl:try with avg() over attribute values, computed -->
    
    <xsl:template name="s-120" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try select="round(avg(account/transaction/abs(@value)))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg() of a conditional value -->
    
    <xsl:template name="s-121" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try select="round(avg(account/transaction/
            (if (xs:date(@date) lt xs:date('2020-01-01')) then +@value else (@value+1))))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Test of xsl:source-document/xsl:try with avg() with a loop used to compute each item -->
    
    <xsl:template name="s-122" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg(BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document/xsl:try calling avg() using //x/y -->
    <!-- Saxon makes this streamable by rewriting //X/Y as .//Y[parent::X] -->
    
    <xsl:template name="s-123" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg(outermost(//DIMENSIONS)!xs:NMTOKENS(.)!xs:decimal(.))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg() with xsl:try: grounded operand, selects nothing -->
    
    <xsl:template name="s-140" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg(data(/*/NOTHING))"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Streaming avg() with xsl:try: striding operand, selects nothing -->
    
    <xsl:template name="s-141" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg(/BOOKLIST/BOOKS/MAGAZINE)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Streaming avg() with xsl:try: crawling operand, selects nothing -->
    
    <xsl:template name="s-142" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg(/*/*/MAGAZINE)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
  <!-- Streaming avg() with xsl:try: climbing operand, selects nothing -->
    
    <xsl:template name="s-143" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try select="avg(//PRICE/../@nothing)"><xsl:catch select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
                                               
    
</xsl:stylesheet>