<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- This test set is derived by direct conversion of si-attribute-A to use elements instead of attributes -->

    
    <!-- Test of xsl:source-document with literal result element -->
    
    <xsl:template name="s-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICE"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, filtered with a motionless predicate -->
    
    <xsl:template name="s-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="./BOOKLIST/BOOKS/ITEM[@CAT='P']/PRICE"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, filtered with a positional predicate -->
    
    <xsl:template name="s-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="./BOOKLIST/BOOKS/ITEM[position() lt 4]/PRICE"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, filtered with a positional predicate -->
    
    <xsl:template name="s-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="./BOOKLIST/BOOKS/*[position() lt 4]/PRICE"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, filtered with a positional predicate -->
    
    <xsl:template name="s-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="/BOOKLIST/BOOKS/*:ITEM[position() lt 4]/PRICE"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, filtered using data() to make it streamable -->
    
    <xsl:template name="s-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PAGES/data()[. &lt; 1000][. &gt; 0]"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, filtered using text() to make it streamable -->
    
    <xsl:template name="s-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="BOOKLIST/BOOKS/ITEM/PAGES/text()[. &lt; 1000][. &gt; 0]"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, filtered using number() to make it streamable -->
    
    <xsl:template name="s-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="/BOOKLIST/BOOKS/ITEM/PAGES/number()[. &lt; 1000][. &gt; 0]"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, selecting nothing  -->
    
    <xsl:template name="s-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="ITEM/PAGES"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:element selecting both streamed nodes and literals -->
    
    <xsl:template name="s-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PAGES/number(), 31, 32"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:element selecting both streamed nodes and literals while also filtering -->
    
    <xsl:template name="s-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="tail(./BOOKLIST/BOOKS/ITEM/PAGES)/number(), 31, 32"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>   
    
    <!-- Test of xsl:source-document with literal result element of a computed value -->
    
    <xsl:template name="s-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a>
          	<xsl:sequence select="for $d in data(outermost(//DIMENSIONS))
                                return let $x := tokenize($d, '\s')!number() 
                                       return round($x[1]*$x[2]*$x[3], 3)"/>
          </a>                             
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element of a computed value -->
    
    <xsl:template name="s-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <a><xsl:sequence select="account/transaction[position() lt 5]/(@value*2)"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element, filtered using copy-of() to make it streamable -->
    
    <xsl:template name="s-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PAGES/copy-of()[. &lt; 1000][. &gt; 0]"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element applied to an attribute -->
    
    <xsl:template name="s-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <a><xsl:sequence select="account/transaction[position() lt 5]/@value"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element over attribute values, filtered -->
    
    <xsl:template name="s-019" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <a><xsl:sequence select="account/transaction/@value[xs:decimal(.) lt -8.00]"/></a>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Test of xsl:source-document with literal result element over attribute values, computed -->
    
    <xsl:template name="s-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <a><xsl:sequence select="account/transaction[position() mod 10 = 1]/abs(@value)"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- xsl:element of a conditional value -->
    
    <xsl:template name="s-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <a>
             <xsl:sequence select="account/transaction/
                (if (xs:date(@date) lt xs:date('2020-01-01')) then +@value else (@value+1))"/>
          </a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- xsl:element with a loop used to compute each item -->
    
    <xsl:template name="s-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.)"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling xsl:element using outermost(//x/y) -->
    
    <xsl:template name="s-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="outermost(//ITEM/DIMENSIONS)!xs:NMTOKENS(.)!xs:decimal(.)"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:element: grounded operand, selects nothing -->
    
    <xsl:template name="s-040" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="data(/*/*/NOTHING)"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:element: striding operand, selects nothing -->
    
    <xsl:template name="s-041" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="/BOOKLIST/BOOKS/MAGAZINE"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:element: crawling operand, selects nothing -->
    <!-- TODO: not guaranteed streamable -->
    
    <xsl:template name="s-042" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="/*/*/MAGAZINE"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:element: climbing operand, selects nothing -->
    
    <xsl:template name="s-043" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="//PRICE/../@nothing"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:element: complex content -->
    
    <xsl:template name="s-046" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a>
            <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
              <xsl:text>[</xsl:text>
              <xsl:value-of select="PRICE"/>
              <xsl:text>]</xsl:text>
            </xsl:for-each>
          </a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:element: attribute is consuming  -->
    
    <xsl:template name="s-047" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a name="{translate(head(//AUTHOR), ' ', '_')}">value</a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:element: attribute is consuming  -->
    
    <xsl:template name="s-048" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a namespace="http://{lower-case(translate(head(//AUTHOR), ' ', '_'))}.com/">value</a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:element: attribute is consuming and sequence-valued -->
    
    <xsl:template name="s-049" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a prices="{outermost(//PRICE)}">value</a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element/@xsl:on-empty (not empty) -->
    
    <xsl:template name="s-050" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:where-populated>
            <a><xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICE/data()"/></a>
          </xsl:where-populated>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element/@xsl:on-empty (empty) -->
    
    <xsl:template name="s-051" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:where-populated>
            <a><xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICETAG"/></a>
          </xsl:where-populated>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element/@xsl:on-empty, with xsl:on-empty consuming (not empty) -->
    
    <xsl:template name="s-052" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a>
            <xsl:sequence select="1 to 10"/>
            <xsl:on-empty select="head(//*)"/>
          </a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element/@xsl:on-empty, with xsl:on-empty consuming (empty) -->
    
    <xsl:template name="s-053" use-when="$RUN">
      <xsl:param name="s" select="20"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
            <xsl:sequence select="$s to 10"/>
            <xsl:on-empty select="head(//TITLE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element/@xsl:on-empty (empty, returns a new element) -->
    
    <xsl:template name="s-054" use-when="$RUN">
      <xsl:variable name="e" as="element()"><e/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS">
          <out>
            <xsl:where-populated>
              <a><xsl:sequence select="ITEM/PRICETAG"/></a>
            </xsl:where-populated>
            <xsl:on-empty select="$e"/>
          </out>
        </xsl:for-each>  
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with literal result element/@xsl:on-empty (empty, type error, returns two elements) -->
    
    <xsl:template name="s-055" use-when="$RUN">
      <xsl:param name="n" select="2"/>
      <xsl:variable name="atts" as="element()*">
        <xsl:for-each select="1 to $n">
          <xsl:element name="n{.}"><xsl:sequence select="."/></xsl:element>
        </xsl:for-each>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS">
          <out>
            <xsl:where-populated>
              <a><xsl:sequence select="ITEM/PRICETAG"/></a>
            </xsl:where-populated>
            <xsl:on-empty select="$atts"/>
          </out>
        </xsl:for-each>  
      </xsl:source-document>
    </xsl:template>
    
    <!-- LRE content cannot contain a function item - error -->
    
    <xsl:template name="s-057" use-when="$RUN">
      <xsl:param name="s" select="false#0"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <a><xsl:sequence select="1, 2, head(//text()), $s"/></a>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- xsl:element content cannot contain a function item - error - recovered -->
    <!-- modified by="Michael Kay" on="2018-06-26" change="Added XTDE0450 which is the correct error code, but left FOTY0013 for old times sake" -->
      
    
    <xsl:template name="s-058" use-when="$RUN">
      <xsl:param name="s" select="false#0"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
            <a><xsl:sequence select="1, 2, head(//text()), $s"/></a>
            <xsl:catch errors="*:FOTY0013 *:XTDE0450">caught</xsl:catch>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- LRE content  = "." -->
    
    <xsl:template name="s-059" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="*">
            <in><xsl:sequence select="."/></in>
          </xsl:for-each>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    
                                              
    
</xsl:stylesheet>